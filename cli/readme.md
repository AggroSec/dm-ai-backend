# Twin Fates — Ironweave System CLI

A Python command-line client for the DM AI backend.

## Setup

```bash
pip install -r requirements.txt
chmod +x dm_ai.py
```

## Configuration

By default connects to `http://localhost:8080`. Override with:

```bash
export DM_AI_URL=http://your-server:8080
```

## Usage

```bash
python dm_ai.py
```

### Commands

| Command | Description |
|---------|-------------|
| `login` | Log in to your account |
| `logout` | Log out and revoke session |
| `new` | Create a new campaign and character |
| `play` | Resume or start a campaign |
| `help` | Show available commands |
| `quit` | Exit the CLI |

## Session Storage

Your JWT and refresh token are stored in `~/.dm_ai_session`. The CLI automatically refreshes your JWT when it expires using the stored refresh token.