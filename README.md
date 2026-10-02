<div align="center">

<h1>🎵 Telegram Web Player</h1>

<p>
  <b>A modern Telegram web music player with real-time synchronized playback, personal playlists, and smart recommendations.</b>
</p>

<p>
  <a href="https://golang.org/">
    <img src="https://img.shields.io/badge/Written%20in-Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Language">
  </a>
  <a href="https://www.docker.com/">
    <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
  </a>
  <a href="https://github.com/AshokShau/telegram-web-player/blob/master/LICENSE">
    <img src="https://img.shields.io/badge/License-GPL%20v3-4bc51d?style=for-the-badge" alt="License">
  </a>
  <a href="https://github.com/AshokShau/telegram-web-player/stargazers">
    <img src="https://img.shields.io/github/stars/FallenProjects/telegram-web-player?style=for-the-badge&color=ffd700&logo=github" alt="Stars">
  </a>
  <a href="https://github.com/AshokShau/telegram-web-player/network/members">
    <img src="https://img.shields.io/github/forks/FallenProjects/telegram-web-player?style=for-the-badge&color=blue&logo=github" alt="Forks">
  </a>
</p>

---

<p align="center">
  Telegram Web Player delivers synchronized, low-latency web playback for chat rooms.<br>
  Engineered with <b>Go</b>, <code>gotdbot</code> (TDLib), WebSockets, MongoDB, and a responsive glassmorphic Web App interface.
</p>

</div>

---

## 🔥 Key Features

### 🎧 Synchronized Web Player Interface
- **Real-Time WebSocket Synchronization**: Instant synchronization of playback position, playing/paused state, track changes, and queue updates across all connected listeners in a room.
- **Pure Web Audio Playback**: Streams audio directly inside browser or Telegram Mini App using HTML5 Web Audio — no native Telegram Voice Chat (VC) connection required.
- **Glassmorphic Responsive UI**: Styled with glassmorphism effects, dynamic album artwork blur background layer, live seek slider, volume level dynamic icons and mobile/desktop responsiveness.
- **Active Listener Roster**: Live display of connected room participants complete with Telegram avatars, names, and admin indicators.

### 📚 Personal Playlists & Queue Management
- **Interactive Playlist Manager**: Create, rename, view, and delete personal custom playlists directly in the WebApp or via bot commands.
- **Playlist Actions**: Add currently playing tracks or search results to custom playlists, reorder songs, queue entire playlists (`+ Queue All`), or force-play playlists with 1 tap.
- **Full Queue Controls**: Reorder queue, skip tracks, seek to timestamps, set loop counts (0–10), or clear remaining queue items.

### 🤖 Smart Autoplay & Mix Recommendations
- **YouTube Mix Engine**: Instantly generate dynamic mixes of related songs based on queries or currently playing tracks via `/mix` or WebApp search.
- **Continuous Autoplay**: Automatically queues recommended songs when the main queue finishes, keeping music playing seamlessly.

### ⏱️ Sleep Timer & Session Protection
- **Custom Sleep Timer**: Built-in sleep timer drawer supporting durations up to 2 hours (15m, 30m, 45m, 1h, 2h). Automatically pauses local audio and closes Telegram Mini App without disrupting other room listeners.
- **Single Active Session Protection**: Prevents duplicate active sessions per user.
- **Listener Grace Timer**: Automatically pauses session after a 40-second grace period when all listeners leave a room, preserving server resources.

### 🔐 Security & Chat Administration
- **Telegram Mini App Security**: Cryptographic `initData` HMAC verification against bot token hash to guarantee authenticated user sessions.
- **Granular Group Controls**: Configurable permissions for play mode (Everyone vs. Admins) and admin controls (Skip, Stop, Seek, Loop, Queue Clear).
- **Authorized Users List**: Grant or revoke specific bot admin privileges per chat with `/auth` and `/removeAuth`.

---

## 📋 Requirements

Before deploying, ensure you have:

1. **Linux Server** (Ubuntu 22.04 LTS or Debian 12 recommended) or a **Docker environment**.
2. **Go 1.26 or higher** (if installing manually without Docker).
3. **MongoDB Database**: Free cluster on [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) or self-hosted MongoDB instance.
4. **Telegram API Credentials**: `API_ID` and `API_HASH` from [my.telegram.org](https://my.telegram.org).
5. **Telegram Bot Token**: HTTP API token generated via [@BotFather](https://t.me/BotFather).
6. **FFmpeg & yt-dlp**: Required for media extraction and stream downloading.

---

## ⚙️ Environment Configuration

<details>
<summary><b>Click to view Environment Variables & Credentials Setup</b></summary>

<br>

Copy `sample.env` to create your environment configuration file:

```bash
cp sample.env .env
```

### Where to get credentials

- **`API_ID` & `API_HASH`**: Log in to [my.telegram.org](https://my.telegram.org) with your Telegram phone number, select **API development tools**, and create an application.
- **`TOKEN`**: Message [@BotFather](https://t.me/BotFather) on Telegram, send `/newbot`, follow the instructions, and copy the bot token provided.
- **`MONGO_URI`**: Register at [MongoDB Atlas](https://www.mongodb.com/cloud/atlas), create a database cluster, go to **Database Access / Network Access** to permit connections, and copy the connection string (`mongodb+srv://...`).
- **`OWNER_ID`**: Send `/id` to [@userinfobot](https://t.me/userinfobot) on Telegram to get your numeric user ID.

### Environment Variables Reference

| Variable              | Required | Default                       | Description                                                               |
|-----------------------|:--------:|-------------------------------|---------------------------------------------------------------------------|
| `API_ID`              | **Yes**  | -                             | Telegram API ID from my.telegram.org.                                     |
| `API_HASH`            | **Yes**  | -                             | Telegram API Hash from my.telegram.org.                                   |
| `TOKEN`               | **Yes**  | -                             | Telegram Bot Token from @BotFather.                                       |
| `OWNER_ID`            | **Yes**  | -                             | Telegram User ID of the bot owner.                                        |
| `MONGO_URI`           | **Yes**  | -                             | MongoDB connection URI string.                                            |
| `PORT`                |    No    | `6060`                        | Web server HTTP port for Web App and WebSockets.                          |
| `API_URL`             |    No    | `https://api.onegrab.fun`     | Downloader API endpoint URL.                                              |
| `API_KEY`             |    No    | -                             | Optional API Key for downloader API.                                      |
| `DL_BOT_TOKEN`        |    No    | -                             | Optional secondary downloader bot token for Telegram file fetching.       |
| `DB_NAME`             |    No    | `Anon`                        | Database name inside MongoDB.                                             |
| `LOGGER_ID`           |    No    | `0`                           | Telegram chat/channel ID where bot startup logs and errors are sent.      |
| `DEFAULT_SERVICE`     |    No    | `youtube`                     | Default search engine for track queries (`youtube` or `spotify`).         |
| `AUTO_PLAY_LIMIT`     |    No    | `10`                          | Maximum number of recommended tracks queued during autoplay.              |
| `SONG_DURATION_LIMIT` |    No    | `3600`                        | Maximum track duration allowed in seconds (default: 1 hour).              |
| `MAX_FILE_SIZE`       |    No    | `524288000`                   | Maximum file download size limit in bytes (default: 500 MB).              |
| `DOWNLOADS_DIR`       |    No    | `downloads`                   | Local temporary directory for media downloads.                            |
| `PROXY`               |    No    | -                             | Optional HTTP/SOCKS proxy URL for external media downloads.               |
| `COOKIES_URL`         |    No    | -                             | Comma-separated HTTP URLs pointing to raw YouTube `cookies.txt` files.    |
| `SUPPORT_GROUP`       |    No    | `https://t.me/FallenSupport`  | Support group URL shown in help menus.                                    |
| `SUPPORT_CHANNEL`     |    No    | `https://t.me/FallenProjects` | Updates channel URL shown in help menus.                                  |
| `START_IMG`           |    No    | (default URL)                 | Direct image URL displayed in `/start` command response.                  |
| `DEVS`                |    No    | -                             | Space or comma separated list of additional developer user IDs.           |

</details>

--- 

<details>
<summary><b>Click to view Domain & SSL Setup (Cloudflare Tunnel, Caddy, Nginx)</b></summary>

<br>

Telegram Mini Apps **require** a valid HTTPS domain (`https://`). Below are the easiest ways to expose your local or VPS web player port (`6060`) securely to a custom domain.

#### Option 1: Cloudflare Tunnel with Docker Compose (Easiest & Free)

Cloudflare Tunnel lets you route traffic from your domain to your local Docker container without opening inbound firewall ports or configuring SSL manually.

1. Go to **Cloudflare Zero Trust Dashboard** -> **Networks** -> **Tunnels** and click **Create a Tunnel**.
2. Name your tunnel, copy the tunnel token provided (`eyJh...`), and add it to your `.env` file:
   ```env
   TUNNEL_TOKEN=eyJh...
   ```
3. Route your hostname (e.g. `music.yourdomain.com`) to HTTP service `tg-web:6060` (or `localhost:6060`).
4. Update your `docker-compose.yml` to include the Cloudflare Tunnel service:

```yaml
services:
  tg-web:
    build: .
    env_file: .env
    ports:
      - "${PORT:-6060}:${PORT:-6060}"
    restart: unless-stopped

  cloudflared:
    image: cloudflare/cloudflared:latest
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      - TUNNEL_TOKEN=${TUNNEL_TOKEN}
```

5. Run `docker compose up -d` and set your Web App URL in BotFather to `https://music.yourdomain.com/room`.

> **Quick Tunnel (Temporary / Testing):**
> If you don't own a domain yet, run `cloudflared tunnel --url http://localhost:6060` to instantly get a temporary `https://xxx.trycloudflare.com` URL.

---

#### Option 2: Caddy Reverse Proxy (Automatic SSL)

Caddy automatically obtains and renews Let's Encrypt TLS/SSL certificates for your domain.

1. Install Caddy on your server:
   ```bash
   sudo apt install -y caddy
   ```
2. Edit `/etc/caddy/Caddyfile`:
   ```caddy
   music.yourdomain.com {
       reverse_proxy localhost:6060
   }
   ```
3. Restart Caddy:
   ```bash
   sudo systemctl restart caddy
   ```

---

#### Option 3: Nginx + Certbot

If you already use Nginx:

1. Create a site configuration (`/etc/nginx/sites-available/tgweb`):
   ```nginx
   server {
       server_name music.yourdomain.com;

       location / {
           proxy_pass http://127.0.0.1:6060;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection "upgrade";
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```
2. Enable site and issue SSL certificate:
   ```bash
   sudo ln -s /etc/nginx/sites-available/tgweb /etc/nginx/sites-enabled/
   sudo certbot --nginx -d music.yourdomain.com
   ```

</details>

<details>
<summary><b>Click to view: Web App Setup Guide in BotFather</b></summary>

<br>

To attach the Web Player interface to your Telegram bot:

```text
/newapp

BotFather:
Alright, a new web app. Which bot will be offering the web app?

@YourMusicBot

BotFather:
Creating a new web app for @YourMusicBot.
Please enter a title for the web app.

SyncTune Player

BotFather:
Please enter a short description of the web app.

Synchronized Web Music Player for Telegram

BotFather:
Please upload a photo, 640x360 pixels.

[ Upload 640x360 image ]

BotFather:
Now upload a demo GIF or send /empty to skip this step.

/empty

BotFather:
No problem, you can always add a GIF later using /editapp.
Now please send me the Web App URL.

https://your-domain.com/room

BotFather:
Good! Now please choose a short name for your web app.
3-30 characters: a-z, A-Z, 0-9, _ or .

- THIS IS IMP DONT CHANGE THIS 
web 

BotFather:
You can now use "web" as the short_name parameter value in Bot API.
Your web app link is:

https://t.me/YourMusicBot/web
```

</details>

---

## 🚀 Deployment

<details>
<summary><b>Docker Deployment (Recommended)</b></summary>

<br>

Docker isolates all required dependencies (Go 1.26, FFmpeg, yt-dlp, Deno, TDLib) inside a container.

#### 1. Install Docker & Docker Compose
On Ubuntu / Debian:

```bash
sudo apt update
sudo apt install -y docker.io docker-compose-v2
sudo systemctl enable --now docker
```

#### 2. Clone Repository & Configure Environment

```bash
git clone https://github.com/AshokShau/telegram-web-player.git
cd telegram-web-player
cp sample.env .env
nano .env
```

Fill in all required variables inside `.env` (`API_ID`, `API_HASH`, `TOKEN`, `OWNER_ID`, `MONGO_URI`), then save and exit (`Ctrl + O`, `Enter`, `Ctrl + X`).

#### 3. Build & Run Container

Using Docker Compose:

```bash
docker compose up -d --build
```

Or using standard Docker CLI:

```bash
docker build -t tgweb .
docker run -d --name tgweb --env-file .env -p 6060:6060 --restart unless-stopped tgweb
```

#### 4. Container Management

- **View Logs**:
  ```bash
  docker compose logs -f
  ```
- **Stop Bot**:
  ```bash
  docker compose down
  ```
- **Restart Bot**:
  ```bash
  docker compose restart
  ```

</details>

<details>
<summary><b>Linux Setup (Ubuntu / Debian)</b></summary>

<br>

#### 1. Install System Dependencies

```bash
sudo apt update
sudo apt install -y ffmpeg curl wget unzip git
```

Install **yt-dlp**:

```bash
sudo wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

Install **Deno** (required for YouTube download challenges):

```bash
curl -fsSL https://deno.land/install.sh | sh
echo 'export DENO_INSTALL="$HOME/.deno"' >> ~/.bashrc
echo 'export PATH="$DENO_INSTALL/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Install **Go** (1.26 or higher required):

```bash
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 2. Clone Repository & Prepare Configuration

```bash
git clone https://github.com/AshokShau/telegram-web-player.git
cd telegram-web-player
cp sample.env .env
nano .env
```

Fill in your configuration settings in `.env`.

#### 3. Download TDLib Library & Build

```bash
# Generate / download required TDLib library
go generate

# Build binary
go build -o tgweb main.go
```

#### 4. Run the Bot

Test run directly in terminal:

```bash
./tgweb
```

#### 5. Background Execution

To keep the bot running after disconnecting from SSH, use `tmux` or `screen`.

##### Option A: Using `tmux`

```bash
# Start new session
tmux new -s tgweb

# Run bot
./tgweb
```
Detach with `Ctrl + B`, then press `D`.
Reattach later: `tmux attach -t tgweb`

##### Option B: Using `screen`

```bash
# Start new screen session
screen -S tgweb

# Run bot
./tgweb
```
Detach with `Ctrl + A`, then press `D`.
Reattach later: `screen -r tgweb`

</details>

---

## 🛠️ Bot Commands

<details>
<summary><b>Click to view Playback Commands</b></summary>

<br>

| Command              | Aliases | Access   | Description                                                                  |
|----------------------|---------|----------|------------------------------------------------------------------------------|
| `/play <query/URL>`  | `/p`    | Everyone | Play audio from YouTube, Spotify, SoundCloud, direct link, or Telegram file. |
| `/fplay <query/URL>` | `/fp`   | Everyone | Force play audio immediately, interrupting current playback.                 |
| `/mix <query/URL>`   | -       | Everyone | Generate a mix of related YouTube tracks based on query or current song.     |
| `/pause`             | -       | Admin    | Pause current room playback.                                                 |
| `/resume`            | -       | Admin    | Resume paused playback.                                                      |
| `/skip`              | -       | Admin    | Skip current track and advance to next in queue.                             |
| `/stop`              | `/end`  | Admin    | Stop playback session and clear room queue.                                  |
| `/seek <seconds>`    | -       | Admin    | Jump to a timestamp in seconds.                                              |
| `/loop <0-10>`       | -       | Admin    | Repeat the current track specified number of times (0 disables).             |
| `/player`            | -       | Everyone | Open the Web App player interface for current chat room.                     |

</details>

<details>
<summary><b>Click to view Queue & Playlist Commands</b></summary>

<br>

| Command                  | Aliases           | Access   | Description                                              |
|--------------------------|-------------------|----------|----------------------------------------------------------|
| `/queue`                 | -                 | Everyone | View current playback queue for the chat room.           |
| `/remove <index>`        | -                 | Admin    | Remove specific track from queue by position index.      |
| `/cplist <name>`         | `/createplaylist` | Everyone | Create a custom personal playlist.                       |
| `/deleteplaylist <name>` | -                 | Everyone | Delete a personal playlist.                              |
| `/addtoplaylist`         | `/addtoplist`     | Everyone | Add current track or replied audio to personal playlist. |
| `/removefromplaylist`    | `/rmplist`        | Everyone | Remove track from personal playlist.                     |
| `/playlistinfo <name>`   | `/plistinfo`      | Everyone | View tracks in a personal playlist.                      |
| `/myplaylists`           | `/myplist`        | Everyone | List all your custom personal playlists.                 |

</details>

<details>
<summary><b>Click to view Admin & Group Setup Commands</b></summary>

<br>

| Command              | Aliases    | Access   | Description                                          |
|----------------------|------------|----------|------------------------------------------------------|
| `/auth <user>`       | `/addAuth` | Admin    | Grant bot admin rights in chat to a user.            |
| `/removeAuth <user>` | `/rmAuth`  | Admin    | Revoke bot admin rights from a user.                 |
| `/authList`          | `/auths`   | Everyone | List authorized users in current chat.               |
| `/settings`          | -          | Owner    | Open interactive settings menu for chat preferences. |
| `/autoplay`          | -          | Admin    | Toggle automatic recommended track queuing.          |
| `/reload`            | -          | Admin    | Refresh chat admin cache.                            |

</details>

<details>
<summary><b>Click to view Owner & Developer Commands</b></summary>

<br>

| Command            | Aliases       | Access   | Description                                             |
|--------------------|---------------|----------|---------------------------------------------------------|
| `/stats`           | -             | Devs     | Display system resource usage and bot statistics.       |
| `/av`              | `/activevc`   | Devs     | View active music room sessions across chats.           |
| `/broadcast <msg>` | `/gCast`      | Owner    | Broadcast message to served chats.                      |
| `/stop_broadcast`  | `/stop_gcast` | Owner    | Cancel active broadcast execution.                      |
| `/logger`          | -             | Devs     | View logging channel status.                            |
| `/privacy`         | -             | Everyone | Display privacy settings info.                          |
| `/sh`              | -             | Devs     | Execute shell command.                                  |
| `/ping`            | -             | Everyone | Check bot latency and uptime status.                    |

</details>

---

## 🔄 Updating the Bot

When new updates are released, update your deployment using the steps below:

### Docker Deployment Update

```bash
cd telegram-web-player
git pull origin master
docker compose down
docker compose up -d --build
```

### Linux Deployment Update

```bash
cd telegram-web-player
# Stop running instance (e.g. exit tmux/screen session)

# Pull latest commits
git pull origin master

# Update Go dependencies and library
go mod download
go generate

# Recompile binary
go build -o tgweb main.go

# Restart process inside tmux or screen
```

---

## ❓ Troubleshooting & FAQ

<details>
<summary><b>YouTube playback fails with 403 Forbidden or Sign-in errors</b></summary>

<br>

- YouTube frequently updates bot detection mechanisms.
- Export raw cookies from your browser (using extensions like *Get cookies.txt LOCALLY*).
- Upload the `cookies.txt` file to a URL or GitHub Gist (raw link) and set `COOKIES_URL` in your `.env`.
</details>

<details>
<summary><b>Audio does not play automatically when opening Web App</b></summary>

<br>

- Mobile operating systems (iOS and Android) require an initial user gesture before playing web audio.
- Click the play button or tap anywhere on the interface to trigger the automatic **Audio Unlocking** mechanism.
</details>

<details>
<summary><b>Duplicate Session warning screen appears</b></summary>

<br>

- Each Telegram user is permitted **one active Web Player connection** at a time to prevent state conflicts.
- If you opened the player on another device or web browser tab, close the previous session and tap **Restart App** on the duplicate session screen.
</details>

---

## 🤝 Contributing

Contributions, issues, and feature suggestions are welcome!

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/amazing-feature`.
3. Commit your changes: `git commit -m 'Add amazing feature'`.
4. Push to the branch: `git push origin feature/amazing-feature`.
5. Open a Pull Request.

---

## 📄 License

This project is licensed under the **GNU General Public License v3.0**. See the [LICENSE](./LICENSE) file for full details.

---

## 💬 Support & Updates

- **Support Group**: [Telegram Support](https://t.me/FallenSupport)
- **Updates Channel**: [Telegram Channel](https://t.me/FallenProjects)

---

## ❤️ Donate

If you find this project useful, consider supporting its development with a donation:

- **GRAM (TON) / USDT-GRAM:** `UQD8rsWDh3VD9pXVNuEbM_rIAKzV07xDhx-gzdDe0tTWGXan`
- **USDT (TRC20):** `TJWZqPK5haSE8ZdSQeWBPR5uxPSUnS8Hcq`
- **Telegram Wallet:** [@Ashokshau](https://t.me/Ashokshau)

Thank you for supporting the project!

---

<p align="center">
  Made with 🖤 by <a href="https://github.com/ashokshau">Ashok Shau</a>
</p>
