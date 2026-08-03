# Persona Control Plane

A minimalistic, light-themed UI for managing Indieclaw personas without code changes.

## 🎨 Features

- **Minimalistic Design**: Clean, beige-themed interface
- **Persona Management**: Create, edit, delete personas
- **TOML Editor**: Full TOML configuration editing
- **Real-time Validation**: Validate TOML syntax before saving
- **Phone Whitelist**: Manage WhatsApp authorized numbers
- **Zero Config**: Automatically finds persona files from gateway-service

## 🚀 Quick Start

### 1. Install Dependencies

```bash
cd persona-control-plane
npm install
```

### 2. Start the Server

```bash
npm start
# or for development
npm run dev
```

The UI will be available at: **http://localhost:3000**

### 3. Open Your Browser

```
http://localhost:3000
```

## 🎯 Usage

### View Personas

The sidebar shows all available personas. Click any persona to view and edit it.

### Edit a Persona

1. Select a persona from the sidebar
2. Edit the TOML configuration in the editor
3. Click "Validate" to check syntax
4. Click "Save" to apply changes

### Create New Persona

1. Click "+ New" in the sidebar
2. Enter a persona name (e.g., `my_coach`)
3. The default template will be created
4. Edit and save your configuration

### Delete Persona

1. Select a persona from the sidebar
2. Click "Delete" to remove it
3. Note: Default persona cannot be deleted

### Validate Configuration

Before saving:
1. Click "Validate" to check TOML syntax
2. Fix any errors shown
3. Click "Save" when valid

## 📂 Configuration File

Persona files are stored as TOML in:

```
gateway-service/config/personas/{persona-name}.toml
```

### Example Persona Structure

```toml
[persona]
name = "Executive Coach Pro"
version = "1.0"

[models]
text_model = "qwen3:8b"
vision_model = "gemma4:e2b"

[personality]
prompt_template = """Your custom prompt here..."""
tone = "coaching"
response_style = "strategic_guidance"

[whatsapp]
allowed_phone_numbers = [
    "91-98765-43210",
    "91-87654-32109",
]
enabled = true
```

## 🎨 Design

The UI features:
- **Light Beige Theme**: Warm, professional color palette (#f5f1e8)
- **Minimalistic Layout**: No unnecessary elements
- **Responsive Design**: Works on mobile, tablet, desktop
- **Dark Text**: High contrast for readability (#5a5450)
- **Smooth Transitions**: Polished interactions

### Color Palette

- Background: `#f5f1e8` (light beige)
- Sidebar: `#faf7f2` (off-white)
- Text: `#5a5450` (dark brown)
- Accent: `#d9cdbf` (warm beige)
- Borders: `#e8dfd6` (light beige)

## 🔧 API Endpoints

### GET `/api/personas`

List all available personas.

**Response:**
```json
{
  "success": true,
  "personas": ["default", "executive_coach", "my_persona"]
}
```

### GET `/api/personas/:name`

Get a specific persona's configuration.

**Response:**
```json
{
  "success": true,
  "name": "executive_coach",
  "content": "[persona]\nname = \"...\"\n..."
}
```

### POST `/api/personas/:name`

Create or update a persona.

**Body:**
```json
{
  "content": "[persona]\nname = \"...\"\n..."
}
```

### DELETE `/api/personas/:name`

Delete a persona (except default).

### POST `/api/personas/:name/validate`

Validate TOML syntax.

**Body:**
```json
{
  "content": "[persona]\n..."
}
```

**Response:**
```json
{
  "success": true,
  "message": "Valid TOML"
}
```

### GET `/api/status`

Get server status and statistics.

**Response:**
```json
{
  "success": true,
  "status": "running",
  "personasDir": "/path/to/personas",
  "personasCount": 3
}
```

## 📋 Integration

The control plane automatically finds and manages personas in:

```
gateway-service/config/personas/
```

It looks in these locations (in order):
1. `../../gateway-service/config/personas`
2. `../gateway-service/config/personas`
3. `./config/personas`

## 🔐 Security

- No authentication (intended for local/trusted networks)
- Cannot delete default persona
- TOML validation on client and server
- File system isolation (only accesses persona files)

## ⚙️ Environment Variables

```bash
PORT=3000          # Server port (default: 3000)
```

Example:
```bash
PORT=4000 npm start
```

## 🛠️ Development

### Structure

```
persona-control-plane/
├── server.js           # Express server with APIs
├── package.json        # Dependencies
├── public/
│   └── index.html      # Single-page app UI
└── README.md          # This file
```

### Technologies

- **Backend**: Express.js, Node.js
- **Frontend**: Vanilla JavaScript, HTML5, CSS3
- **Config**: TOML parsing

### Adding Features

To add new functionality:

1. **Backend**: Add endpoint in `server.js`
2. **Frontend**: Add API call and UI in `public/index.html`
3. **Styling**: Update CSS in `<style>` tag

## 📞 Troubleshooting

### Port Already in Use

```bash
# Use different port
PORT=4000 npm start
```

### Cannot Find Personas Directory

The server logs the directory path on startup:
```
📂 Using personas directory: /path/to/gateway-service/config/personas
```

If it fails to find the directory, ensure:
1. Gateway service is in expected location
2. `config/personas/` directory exists
3. At least `default.toml` is present

### Changes Not Taking Effect

1. Changes in UI save to disk immediately
2. Orchestrator/Bot must be restarted to load new config
3. Use `PERSONA_NAME` env var to test different personas

## 📚 Related Documentation

- **Persona Configuration**: `gateway-service/config/personas/README.md`
- **Orchestrator Setup**: `PERSONA_QUICKSTART.md`
- **WhatsApp Integration**: `approach-road/wa-echo-loop/PERSONA_INTEGRATION.md`

## 🎓 Example Workflow

### 1. Start Control Plane

```bash
cd persona-control-plane
npm install
npm start
```

### 2. Open Browser

```
http://localhost:3000
```

### 3. Create New Persona

- Click "+ New"
- Name: `coaching_style_1`
- Template appears automatically

### 4. Customize

Edit the TOML:
- Change prompt
- Add client phone numbers
- Adjust tone and style

### 5. Save

- Click "Validate"
- Click "Save"

### 6. Test

In another terminal:
```bash
# Start orchestrator with new persona
PERSONA_NAME=coaching_style_1 go run ./cmd/orchestrator/main.go

# Start WhatsApp bot with same persona
cd approach-road/wa-echo-loop
PERSONA_NAME=coaching_style_1 node app.js
```

### 7. Done!

Now orchestrator and bot use the new persona configuration.

---

**Happy persona managing! 🎨**
