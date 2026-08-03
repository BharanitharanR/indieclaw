const express = require('express');
const fs = require('fs');
const path = require('path');
const bodyParser = require('body-parser');
const toml = require('toml');

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(bodyParser.json({ limit: '10mb' }));
app.use(bodyParser.urlencoded({ limit: '10mb', extended: true }));
app.use(express.static('public'));

// Find personas directory (look in multiple locations)
const possiblePaths = [
  path.join(__dirname, '../../gateway-service/config/personas'),
  path.join(__dirname, '../gateway-service/config/personas'),
  path.join(__dirname, 'config/personas'),
];

let personasDir = null;
for (const dirPath of possiblePaths) {
  if (fs.existsSync(dirPath)) {
    personasDir = dirPath;
    break;
  }
}

if (!personasDir) {
  console.error('❌ Could not find personas directory in any of:', possiblePaths);
  process.exit(1);
}

console.log(`📂 Using personas directory: ${personasDir}`);

// ============ API Routes ============

// GET /api/personas - List all personas
app.get('/api/personas', (req, res) => {
  try {
    const files = fs.readdirSync(personasDir);
    const personas = files
      .filter(f => f.endsWith('.toml'))
      .map(f => f.replace('.toml', ''));
    res.json({ success: true, personas });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// GET /api/personas/:name - Get specific persona
app.get('/api/personas/:name', (req, res) => {
  try {
    const filePath = path.join(personasDir, `${req.params.name}.toml`);
    if (!fs.existsSync(filePath)) {
      return res.status(404).json({ success: false, error: 'Persona not found' });
    }
    const content = fs.readFileSync(filePath, 'utf-8');
    res.json({ success: true, name: req.params.name, content });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// POST /api/personas/:name - Create/update persona
app.post('/api/personas/:name', (req, res) => {
  try {
    const { content } = req.body;
    if (!content) {
      return res.status(400).json({ success: false, error: 'Content is required' });
    }

    // Validate TOML
    try {
      toml.parse(content);
    } catch (tomlError) {
      return res.status(400).json({ success: false, error: `Invalid TOML: ${tomlError.message}` });
    }

    const filePath = path.join(personasDir, `${req.params.name}.toml`);
    fs.writeFileSync(filePath, content, 'utf-8');
    res.json({ success: true, message: `Persona "${req.params.name}" saved` });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// DELETE /api/personas/:name - Delete persona
app.delete('/api/personas/:name', (req, res) => {
  try {
    if (req.params.name === 'default') {
      return res.status(400).json({ success: false, error: 'Cannot delete default persona' });
    }

    const filePath = path.join(personasDir, `${req.params.name}.toml`);
    if (!fs.existsSync(filePath)) {
      return res.status(404).json({ success: false, error: 'Persona not found' });
    }

    fs.unlinkSync(filePath);
    res.json({ success: true, message: `Persona "${req.params.name}" deleted` });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// POST /api/personas/:name/validate - Validate TOML
app.post('/api/personas/:name/validate', (req, res) => {
  try {
    const { content } = req.body;
    toml.parse(content);
    res.json({ success: true, message: 'Valid TOML' });
  } catch (error) {
    res.status(400).json({ success: false, error: `Invalid TOML: ${error.message}` });
  }
});

// GET /api/status - Server status
app.get('/api/status', (req, res) => {
  res.json({
    success: true,
    status: 'running',
    personasDir,
    personasCount: fs.readdirSync(personasDir).filter(f => f.endsWith('.toml')).length,
  });
});

// ============ Server Start ============

app.listen(PORT, () => {
  console.log(`\n📝 Persona Control Plane is running!`);
  console.log(`🌐 Open your browser: http://localhost:${PORT}`);
  console.log(`📂 Managing personas in: ${personasDir}\n`);
});
