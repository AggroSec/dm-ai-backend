#!/usr/bin/env python3
"""
Twin Fates — Ironweave System
DM AI Command Line Interface
"""

from getpass import getpass
import json
import os
import sys
import requests
import textwrap

# ─────────────────────────────────────────────
# Config
# ─────────────────────────────────────────────

BASE_URL = os.getenv("DM_AI_URL", "http://localhost:8080")
SESSION_FILE = os.path.expanduser("~/.dm_ai_session")
CREATION_COMPLETE_SIGNAL = "CHARACTER_CREATION_COMPLETE"
MIN_PASSWORD_LENGTH = 15


class SessionExpiredError(Exception):
    """Raised when there's no valid session (never logged in, or refresh failed).
    Caught in the main loop so we route back to login instead of killing the CLI."""
    pass


# ─────────────────────────────────────────────
# Session management
# ─────────────────────────────────────────────

def load_session():
    if not os.path.exists(SESSION_FILE):
        return {}
    try:
        with open(SESSION_FILE, "r") as f:
            return json.load(f)
    except Exception:
        return {}


def save_session(data: dict):
    with open(SESSION_FILE, "w") as f:
        json.dump(data, f, indent=2)


def clear_session():
    if os.path.exists(SESSION_FILE):
        os.remove(SESSION_FILE)


def get_token() -> str | None:
    return load_session().get("token")


def get_refresh_token() -> str | None:
    return load_session().get("refresh_token")


def refresh_jwt() -> bool:
    refresh_token = get_refresh_token()
    if not refresh_token:
        return False
    resp = requests.post(f"{BASE_URL}/auth/refresh", json={"refresh_token": refresh_token})
    if resp.status_code == 200:
        data = resp.json()
        session = load_session()
        session["token"] = data["jwt"]
        save_session(session)
        return True
    return False


# ─────────────────────────────────────────────
# HTTP helpers
# ─────────────────────────────────────────────

def auth_headers() -> dict:
    token = get_token()
    if not token:
        raise SessionExpiredError("Not logged in.")
    return {"Authorization": f"Bearer {token}"}


def api_get(path: str) -> dict | list | None:
    resp = requests.get(f"{BASE_URL}{path}", headers=auth_headers())
    if resp.status_code == 401:
        if refresh_jwt():
            resp = requests.get(f"{BASE_URL}{path}", headers=auth_headers())
        else:
            clear_session()
            raise SessionExpiredError("Session expired.")
    if not resp.ok:
        print_error(f"Request failed: {resp.json().get('error', resp.text)}")
        return None
    return resp.json()


def api_post(path: str, body: dict, auth: bool = True) -> dict | None:
    headers = auth_headers() if auth else {}
    resp = requests.post(f"{BASE_URL}{path}", json=body, headers=headers)
    if resp.status_code == 401 and auth:
        if refresh_jwt():
            resp = requests.post(f"{BASE_URL}{path}", json=body, headers=auth_headers())
        else:
            clear_session()
            raise SessionExpiredError("Session expired.")
    if not resp.ok:
        print_error(f"Request failed: {resp.json().get('error', resp.text)}")
        return None
    return resp.json()


def api_stream_combat(path: str, body: dict) -> tuple[dict | None, bool]:
    """
    POST to path with SSE streaming for combat only.
    Prints narrative chunks as they arrive.
    Returns (meta, had_error). meta is a dict containing combat_id, combat_ended, etc.
    had_error is True if the stream failed or the server sent an "event: error".
    """
    headers = auth_headers()
    try:
        resp = requests.post(
            f"{BASE_URL}{path}",
            json=body,
            headers=headers,
            stream=True,
        )
    except requests.exceptions.RequestException as e:
        print_error(f"Request failed: {e}")
        return None, True

    if resp.status_code == 401:
        if refresh_jwt():
            try:
                resp = requests.post(
                    f"{BASE_URL}{path}",
                    json=body,
                    headers=auth_headers(),
                    stream=True,
                )
            except requests.exceptions.RequestException as e:
                print_error(f"Request failed: {e}")
                return None, True
        else:
            clear_session()
            raise SessionExpiredError("Session expired.")

    if not resp.ok:
        print_error(f"Request failed: {resp.status_code}")
        return None, True

    resp.encoding = 'utf-8'
    meta = None
    had_error = False
    current_event = None
    print()
    print_divider()

    for raw_line in resp.iter_lines(decode_unicode=True):
        if not raw_line:
            current_event = None
            continue

        if raw_line.startswith("event:"):
            current_event = raw_line[len("event:"):].strip()
            continue

        if raw_line.startswith("data:"):
            data = raw_line[len("data:"):].strip()

            if current_event == "meta":
                try:
                    meta = json.loads(data)
                except json.JSONDecodeError as e:
                    print_error(f"Could not parse server response ({e}). Raw data: {data[:200]}")
                    had_error = True

            elif current_event == "error":
                print_error(f"Server error: {data}")
                had_error = True

            else:
                # narrative chunk — unescape the literal \n markers used for
                # multi-line narrate_combat text, then print as it arrives
                data = data.replace("\\n", "\n")
                for paragraph in data.split("\n"):
                    if paragraph.strip():
                        print(textwrap.fill(paragraph.strip(), width=70))
                    else:
                        print()
    print_divider()
    print()

    return meta, had_error


# ─────────────────────────────────────────────
# Display helpers
# ─────────────────────────────────────────────

DIVIDER = "─" * 60


def print_divider():
    print(DIVIDER)


def print_error(msg: str):
    print(f"\n[!] {msg}\n")


def print_dm(msg: str):
    print()
    print_divider()
    for paragraph in msg.split("\n"):
        if paragraph.strip():
            print(textwrap.fill(paragraph.strip(), width=70))
        else:
            print()
    print_divider()
    print()


def print_system(msg: str):
    print(f"\n[*] {msg}\n")


# ─────────────────────────────────────────────
# Character creation loop
# ─────────────────────────────────────────────

def run_character_creation(campaign_id: str, character_id: str, first_message: str | None = None):
    """Run the character creation conversation loop."""
    print_system("Entering character creation. Type 'quit' to exit and resume later.")
    print_divider()

    first_turn = True
    while True:
        if first_turn and first_message:
            player_input = first_message
            first_turn = False
        else:
            first_turn = False
            player_input = input("You: ").strip()
            if not player_input:
                continue
            if player_input.lower() in ("quit", "exit"):
                print_system("Exiting character creation. Resume with 'play'.")
                return False

        action_resp = api_post("/ai/action", {
            "campaign_id": campaign_id,
            "character_id": character_id,
            "message": player_input,
            "character_creation": True,
        })
        if not action_resp:
            continue

        message = action_resp.get("message", "")

        if CREATION_COMPLETE_SIGNAL in message:
            message = message.replace(CREATION_COMPLETE_SIGNAL, "").strip()
            print_dm(message)
            print_system("Character creation complete! Entering the world...")
            return True

        print_dm(message)

    return False


# ─────────────────────────────────────────────
# Commands
# ─────────────────────────────────────────────

def cmd_register():
    print("\n=== Register ===")
    username = input("Choose a username: ").strip()
    password = getpass("Choose a password: ").strip()
    while len(password) < MIN_PASSWORD_LENGTH:
        print_error(f"Password must be at least {MIN_PASSWORD_LENGTH} characters long.")
        password = getpass("Choose a password: ").strip()
    confirm = getpass("Confirm password: ").strip()

    if password != confirm:
        print_error("Passwords do not match.")
        return

    resp = api_post("/auth/register", {"username": username, "password": password}, auth=False)
    if not resp:
        return

    print_system(f"Account created successfully. Welcome, {resp['username']}!")

    # log them in right away so they don't have to re-enter credentials
    login_resp = api_post("/auth/login", {"username": username, "password": password}, auth=False)
    if not login_resp:
        print_system("Registration succeeded, but automatic login failed. Please run 'login' manually.")
        return

    save_session({
        "token": login_resp["token"],
        "refresh_token": login_resp["refresh_token"],
        "user_id": login_resp["user_id"],
    })
    print_system("You're now logged in. Type 'new' to start your first campaign.")


def cmd_login():
    print("\n=== Login ===")
    username = input("Username: ").strip()
    password = getpass("Password: ").strip()

    resp = api_post("/auth/login", {"username": username, "password": password}, auth=False)
    if not resp:
        return

    save_session({
        "token": resp["token"],
        "refresh_token": resp["refresh_token"],
        "user_id": resp["user_id"],
    })
    print_system(f"Logged in successfully. Welcome, {username}!")


def cmd_logout():
    refresh_token = get_refresh_token()
    if refresh_token:
        api_post("/auth/logout", {"refresh_token": refresh_token})
    clear_session()
    print_system("Logged out successfully.")


def cmd_new_campaign():
    print("\n=== New Campaign ===")
    name = input("Campaign name: ").strip()
    theme = input("Campaign theme (describe the world/setting): ").strip()
    character_name = input("Your character's name: ").strip()

    resp = api_post("/campaigns", {
        "name": name,
        "theme": theme,
        "character_name": character_name,
    })
    if not resp:
        return

    campaign_id = resp["campaign_id"]
    character_id = resp["character_id"]

    print_system(f"Campaign '{name}' created!")

    first_message = f"Hello! I'm ready to create my character. My name is {character_name}."
    completed = run_character_creation(campaign_id, character_id, first_message)

    if completed:
        run_narrative_loop(campaign_id, character_id)


def cmd_play():
    print("\n=== Your Campaigns ===")
    campaigns = api_get("/campaigns")
    if not campaigns:
        return

    if not campaigns:
        print_system("No campaigns found. Use 'new' to create one.")
        return

    for i, c in enumerate(campaigns):
        party_size = len(c.get("party", []))
        creation_status = "character creation pending" if not c.get("character_creation_complete") else "active"
        print(f"  [{i + 1}] {c['name']} — {creation_status} | {party_size} character(s)")
        print(f"      Theme: {c['theme'][:60]}{'...' if len(c['theme']) > 60 else ''}")
        print()

    choice = input("Select a campaign (number): ").strip()
    try:
        idx = int(choice) - 1
        if idx < 0 or idx >= len(campaigns):
            raise ValueError
    except ValueError:
        print_error("Invalid selection.")
        return

    campaign = campaigns[idx]
    campaign_id = campaign["campaign_id"]
    party = campaign.get("party", [])

    if not party:
        print_error("No character found in this campaign.")
        return

    character_id = party[0]["character_id"]

    if not campaign.get("character_creation_complete"):
        print_system(f"Resuming character creation for campaign: {campaign['name']}")
        completed = run_character_creation(campaign_id, character_id)
        if not completed:
            return

    print_system(f"Entering campaign: {campaign['name']}")
    run_narrative_loop(campaign_id, character_id)


def run_narrative_loop(campaign_id: str, character_id: str, combat_id: str | None = None):
    """Main narrative/combat conversation loop."""
    print_system("Type your actions. 'quit' to exit.")
    print_divider()

    while True:
        player_input = input("You: ").strip()
        if not player_input:
            continue
        if player_input.lower() in ("quit", "exit"):
            print_system("Saving your progress. See you next time, adventurer.")
            return

        body = {
            "campaign_id": campaign_id,
            "character_id": character_id,
            "message": player_input,
        }
        if combat_id:
            body["combat_id"] = combat_id

        # combat uses SSE streaming, narrative uses regular JSON
        if combat_id:
            meta, had_error = api_stream_combat("/ai/action", body)

            if had_error:
                # Can't trust combat_id is still valid after any stream failure —
                # drop back to narrative mode instead of repeating the same failure forever.
                combat_id = None
                print_system("Something went wrong resolving that action. Returning to narrative mode — please try again.")
                continue

            if meta is None:
                continue

            new_combat_id = meta.get("combat_id")

            if new_combat_id and not combat_id:
                combat_id = new_combat_id
                print_system("⚔  Combat has begun!")

            if meta.get("combat_ended"):
                combat_id = None
                print_system("Combat has ended. Returning to narrative mode.")

        else:
            action_resp = api_post("/ai/action", body)
            if not action_resp:
                continue

            message = action_resp.get("message", "")
            new_combat_id = action_resp.get("combat_id")

            if new_combat_id:
                combat_id = new_combat_id
                print_system("⚔  Combat has begun!")

            print_dm(message)


def cmd_help():
    print("""
Twin Fates — Ironweave System CLI
──────────────────────────────────
  register  Create a new account
  login     Log in to your account
  logout    Log out and revoke session
  new       Create a new campaign and character
  play      Resume or start a campaign
  help      Show this help message
  quit      Exit the CLI
""")


# ─────────────────────────────────────────────
# Main loop
# ─────────────────────────────────────────────

def main():
    print("""
╔══════════════════════════════════════════╗
║   Twin Fates — Ironweave System          ║
║   AI Dungeon Master                      ║
╚══════════════════════════════════════════╝
""")

    session = load_session()
    if session.get("token"):
        print_system("Session found. Type 'play' to continue your adventure or 'help' for commands.")
    else:
        print_system("Welcome! Type 'register' to create an account, 'login' if you already have one, or 'help' for commands.")

    commands = {
        "register": cmd_register,
        "login": cmd_login,
        "logout": cmd_logout,
        "new": cmd_new_campaign,
        "play": cmd_play,
        "help": cmd_help,
    }

    while True:
        try:
            raw = input("> ").strip().lower()
            if not raw:
                continue
            if raw in ("quit", "exit"):
                print_system("Farewell, adventurer.")
                sys.exit(0)
            if raw in commands:
                commands[raw]()
            else:
                print_error(f"Unknown command: '{raw}'. Type 'help' for available commands.")
        except SessionExpiredError:
            print_error("Your session has expired or you're not logged in. Please log in again.")
            cmd_login()
        except KeyboardInterrupt:
            print("\n")
            print_system("Farewell, adventurer.")
            sys.exit(0)
        except EOFError:
            sys.exit(0)


if __name__ == "__main__":
    main()