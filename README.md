# ⚔️ Twin Fates — Ironweave System

A solo-playable, AI-driven tabletop RPG. An AI model sits in the Dungeon Master's chair — narrating the world, voicing every NPC, and deciding what the story needs next — while a Go backend quietly owns every number behind the curtain: HP, dice, damage, inventory, XP, the works. A Python CLI puts it in your hands.

This is a boot.dev capstone project, built solo, end to end — backend, game engine, AI orchestration layer, and CLI client.

🧙 Step into a world. Roll the dice. Let the story go somewhere you didn't plan.

## 🏆 The Golden Rule

If it affects a **number**, Go owns it. If it affects a **sentence**, the AI owns it.

The AI never decides how much damage lands, whether a character survives a hit, or what a d20 rolled — those are deterministic and enforced server-side, because letting an LLM "decide" the math would make the game inconsistent and trivially breakable. Instead, the AI narrates, roleplays every NPC, and makes narrative judgment calls (does this moment deserve a roll? does this loot fit this NPC?), then reaches for a fixed toolbox to actually apply outcomes. Go validates and executes every one of those calls against real game state in Postgres. The AI tells the story; it never gets to cheat the dice.

## ✨ Features

- 🎭 **AI Dungeon Master** via OpenRouter, with separate narrative and combat system prompts — and a distinct toolbox scoped to each mode (narrative, combat, character creation), so the AI is never sitting there holding tools that have nothing to do with what it's currently doing.
- ⚔️ **Turn-based tactical combat** — AP/HP/WP resource management, status effects, per-class skill trees, all validated and resolved server-side, not narrated into existence.
- 📖 **Data-driven classes and fates** — classes, skill trees, and narrative "fate" bonuses/drawbacks live as JSON under `data/`, not hardcoded, so adding new content is a content change, not a code change.
- 🎲 **Guided character creation** — a real, step-by-step AI-led flow (race → class → fate → stats → starting gear), backed by actual tool calls, not free-form text that may or may not stick.
- 🧠 **Rolling narrative memory** — as a campaign grows, older messages get periodically folded into a running narrative summary, so the AI holds onto long-term continuity without the raw message history blowing out its context window. Summarization is append-only: each pass writes one *new* paragraph onto the story so far, rather than regenerating the whole thing — so earlier chapters never get quietly reworded or lossily compressed the longer you play.
- 🔀 **Model fallback chains** — narrative and combat each get their own configured model, and each one lists the *other* as backup before finally falling through to OpenRouter's free-model auto-router, so one congested or rate-limited model doesn't take the whole session down mid-story.
- 🔐 **JWT auth with refresh tokens**, full campaign/character persistence, and a complete REST API underneath it all.

## 🗺️ The World, the Rules, and Building a Character

Every campaign starts with a **theme** — a setting the AI builds the whole story around. Out of the box this project ships tuned for an arena-fighting theme (step into the ring, roleplay and shop between bouts), but the theme is just a prompt-level setting, so you can point it anywhere: a haunted frontier, a heist ring, a war-torn kingdom. The narrative/combat split and every mechanical system underneath work the same no matter what story you tell.

**Classes** — three domains, three very different playstyles:

- 🛡️ **Warrior** *(physical)* — hits hard, takes punishment, outlasts everything. Skills like Power Strike and Adrenaline reward committing to the fight, with talent branches like **Walking Armory** and **Weapon Master** to specialize further.
- ✒️ **Runeblade** *(hybrid)* — combines arcane power with a blade through Etched Runes, sketching magic into the air and detonating it on a physical hit. Branches into paths like **Blood** and **Lightning**.
- 🔮 **Seer** *(magical)* — an oracle who reads the battlefield, curses enemies, shields allies, and deals steady magic damage from range — naturally squishy, more skills per level to compensate. Branches into **Hexblade** and **Shaman**.

**Fates** — every character picks one Driving Fate (a bonus) and one Binding Fate (a drawback) from their class's domain. A Warrior might become **Ironborn** (+5 Fortitude, a body that refuses to quit), a Seer might be an **Arcane Vessel** (+5 Willpower, an innate well of power), and a Runeblade might be **Marked by War** (+4 Strength/+4 Fortitude, at their best when the fighting's fiercest) — balanced against a real cost elsewhere. Fates are locked in at creation and shape a character's identity beyond just a stat block.

**Stats** — six stats, all governing something you'll actually feel at the table:

| Stat | Governs |
|---|---|
| Strength | Physical damage, physical checks |
| Dexterity | Max AP per turn, contributes to AC, agility checks |
| Fortitude | Max HP, contributes to AC (more than Dexterity does), endurance checks |
| Willpower | WP pool, magic damage, magical checks |
| Alacrity | WP regeneration, quick-thinking/fast-talk checks |
| Wisdom | AP overcap, magic defense, perception/insight checks |

A Seer who dumps Fortitude and Dexterity is a glass cannon — valid, but the game makes sure you choose that on purpose, not by accident.

## 🤖 A Note on AI Model Quality

This project runs on free OpenRouter models by design (see [Configuration](#getting-started)), and model quality **varies a lot** between them — sometimes dramatically, even between two models that both claim tool-calling support. Narrative flavor and, especially, combat reliability (correct tool sequencing, respecting AP/HP rules, not hallucinating mechanics) have differed noticeably from one free model to the next in testing, and the "best" model can shift over time as providers update or retire them.

If combat starts ignoring rules, dishing out inconsistent damage, or the DM seems to be "forgetting" tools it should be calling — try swapping `OPENROUTER_MODEL` / `OPENROUTER_COMBAT_MODEL` before assuming the backend is broken. The tool-calling loop, validation, and state management are all model-agnostic; the model itself is doing all the actual reasoning about *when* to call what. Combat asks a lot more of a model than freeform narrative does, which is exactly why narrative and combat get independently configurable models instead of one shared setting.

## 🛠️ Tech Stack

- **Backend:** Go, `net/http` (standard library router)
- **Database:** PostgreSQL, queries via [sqlc](https://sqlc.dev), migrations via [goose](https://github.com/pressly/goose)
- **AI:** [OpenRouter](https://openrouter.ai) (model-agnostic chat completion + tool calling)
- **CLI Client:** Python 3, `requests`
- **Auth:** JWT (access + refresh tokens)

## 🚀 Getting Started

### Prerequisites

- Go (see `go.mod` for version)
- PostgreSQL (via Docker or local install)
- [goose](https://github.com/pressly/goose) for migrations
- [sqlc](https://sqlc.dev) if you plan to modify queries
- An [OpenRouter](https://openrouter.ai) API key (free tier works — see the model quality note above)
- Python 3 + `pip` for the CLI

### Setup

1. **Configure your environment:**

   ```bash
   cp .env.example .env
   ```

   Fill in `.env` with your own values. At minimum you'll need a real `DB_URL`, a generated `JWT_SECRET` (`openssl rand -hex 32`), an `INTERNAL_SECRET`, and your `OPENROUTER_API_KEY`. `OPENROUTER_MODEL` and `OPENROUTER_COMBAT_MODEL` can point at any OpenRouter model ID.

2. **Start Postgres:**

   ```bash
   docker compose up -d
   ```

3. **Run migrations:**

   ```bash
   make migrate-up
   ```

4. **Run the server:**

   ```bash
   make run
   ```

   Starts on the port set by `PORT` in `.env` (default `8080`).

### Other useful Make targets

| Command | Description |
|---|---|
| `make run` | Run the server |
| `make run-log` | Run the server, also tee output to `logs/server.log` |
| `make build` | Build a binary to `bin/server` |
| `make migrate-up` / `make migrate-down` / `make migrate-status` | Manage database schema |
| `make generate` | Regenerate `sqlc` code after changing a query |
| `make test` | Run Go tests |
| `make dbcon` | Open a `psql` shell against the configured database |

### Playing via the CLI

```bash
cd cli
pip install -r requirements.txt
python dm_ai.py
```

By default the CLI connects to `http://localhost:8080`; override with `export DM_AI_URL=http://your-server:8080`. `register` to create an account (or `login` if you already have one), then `new` to roll up a character and start your first campaign. See `cli/readme.md` for the full command list.

## 📡 API Overview

All endpoints are JSON over HTTP unless noted. Routes marked **auth** require a `Bearer` JWT; routes marked **internal** require the internal secret and exist for internal testing, not end users.

| Method & Path | Description |
|---|---|
| `GET /health` | Health check |
| `POST /auth/register` | Create an account |
| `POST /auth/login` | Log in, receive JWT + refresh token |
| `POST /auth/refresh` | Exchange a refresh token for a new JWT |
| `POST /auth/logout` | Revoke a refresh token |
| `POST /characters` **(auth)** | Create a character |
| `GET /characters` **(auth)** | List your characters |
| `GET /characters/{id}` | Get a character by ID |
| `DELETE /characters/{id}` **(auth)** | Delete a character |
| `POST /campaigns` **(auth)** | Create a campaign |
| `GET /campaigns` **(auth)** | List your campaigns |
| `POST /ai/action` **(auth)** | Send a player message; routes to narrative, character creation, or combat (SSE stream) depending on request state |
| `GET /combat/{id}` **(auth)** | Get an active combat session's state |
| `GET /characters/{id}/status_effects` **(auth)** | Get a character's active status effects |
| `POST /rolls` | Roll dice (used internally by the AI tool layer, also callable directly) |

Internal-only endpoints (`requireInternal`) cover direct combat/status-effect manipulation and AI test endpoints used during development — these exist to exercise the game engine and tool-calling loop without needing a full campaign flow.

## 📁 Project Structure

```
dm-ai-backend/
├── cli/               Python CLI client
├── data/              Data-driven class/fate definitions (JSON)
├── internal/
│   ├── ai/            OpenRouter client, prompts, tool definitions, dispatcher, summarization
│   ├── auth/          Password hashing, JWT, refresh tokens
│   ├── config/        Environment-based configuration
│   ├── database/      sqlc-generated queries
│   ├── game/          Core game logic: combat, stats, inventory, leveling, dice
│   └── server/        HTTP handlers and routing
├── sql/
│   ├── queries/       Hand-written SQL queries (sqlc input)
│   └── schema/        Goose migrations
└── main.go
```

## 🔭 Future Work

Things on the roadmap beyond the capstone, roughly in the order they'd get tackled(order not set in stone):

- 📓 **Player journaling** — a lightweight, player-facing log of key moments, meant to reinforce long-term memory alongside the rolling summary, without the cost of full semantic search.
- 📺 **Quick-test embedding** — a browser/TV-friendly embed (e.g. via `ttyd`) for fast playtesting without needing a full client install.
- 🎯 **Smarter summarization triggers** — swapping the flat message-count threshold for something token-aware, since combat and narrative generate very different token densities per message.
- 🧑‍🤝‍🧑 **Multiplayer / couch co-op** — the combat engine is already party-aware under the hood; this is the natural next branch (kept separate from the free base).
- 🖥️ **A real front-end** — something beyond the CLI, whether a lightweight web client or a Godot-based app, so it's easier to hand to someone who isn't comfortable in a terminal.
- 🧬 **Embeddings / RAG** — proper semantic retrieval over full campaign history, for near-complete long-term recall once the free-tier-friendly approaches above stop being enough.
- 📜 **Licensing follow-up** — exploring dual-licensing (AGPL-3.0 public base + a separate commercial license) if this grows past a pet project.

## 📜 License

Licensed under AGPL-3.0. See `LICENSE`.