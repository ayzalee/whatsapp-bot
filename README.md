# whatsapp-bot

A WhatsApp bot built in Go using [whatsmeow](https://github.com/tulir/whatsmeow). Connects via phone-number pairing (no QR scan), persists sessions in SQLite or PostgreSQL, and supports an extensible command plugin system.

## Installation

Pick the script for your platform and run it with elevated privileges. The script handles everything — Go, Git, cloning, building, and PATH setup.

**Windows** (PowerShell as Administrator)

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force
irm https://raw.githubusercontent.com/zaelix/whatsapp-bot/master/scripts/install.ps1 | iex
```

**Linux**

```bash
sudo bash <(curl -fsSL https://raw.githubusercontent.com/zaelix/whatsapp-bot/master/scripts/install-linux.sh)
```

**macOS**

```bash
sudo bash <(curl -fsSL https://raw.githubusercontent.com/zaelix/whatsapp-bot/master/scripts/install-mac.sh)
```

Once complete you will see:

```bash
zaelix --phone-number <international-format-number>
```

A pairing code will be printed. On your phone go to **WhatsApp → Linked Devices → Link a Device → Link with phone number instead** and enter the code.

## License

See [LICENSE](LICENSE).
