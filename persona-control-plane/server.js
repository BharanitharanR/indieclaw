const express = require('express');
const fs = require('fs');
const path = require('path');
const bodyParser = require('body-parser');
const toml = require('toml');
const tomlStringify = require('@iarna/toml').stringify;

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

// ============ PERSONA SCHEMA VALIDATION ============

const VALID_MODES = ['inquiry', 'exploration', 'decision', 'redirect', 'escalate', 'research', 'education', 'clarify', 'reframing'];
const VALID_GATE_TYPES = ['has_element', 'ratio_check', 'length_check', 'confidence_check', 'pattern_check'];

function validatePersonaSchema(config) {
  const errors = [];

  // Check top-level fields
  if (!config.name) errors.push('Missing required field: name');
  if (!config.version) errors.push('Missing required field: version');
  if (typeof config.version !== 'number') errors.push('version must be a number');

  // Check capabilities
  if (config.capabilities) {
    if (!Array.isArray(config.capabilities.can_handle)) {
      errors.push('capabilities.can_handle must be an array');
    }
    if (!Array.isArray(config.capabilities.cannot_handle)) {
      errors.push('capabilities.cannot_handle must be an array');
    }
  } else {
    errors.push('Missing capabilities section');
  }

  // Check intents
  if (config.intents && typeof config.intents === 'object') {
    Object.entries(config.intents).forEach(([intentName, rule]) => {
      if (!rule.mode) {
        errors.push(`Intent "${intentName}" missing required field: mode`);
      } else if (!VALID_MODES.includes(rule.mode)) {
        errors.push(`Intent "${intentName}" has invalid mode "${rule.mode}". Valid modes: ${VALID_MODES.join(', ')}`);
      }
      if (!rule.depth) {
        errors.push(`Intent "${intentName}" missing required field: depth`);
      }
      if (typeof rule.confidence_min !== 'number') {
        errors.push(`Intent "${intentName}" confidence_min must be a number`);
      }
    });
  } else {
    errors.push('Missing or invalid intents section');
  }

  // Check response_rules
  if (config.response_rules && typeof config.response_rules === 'object') {
    Object.entries(config.response_rules).forEach(([modeName, rule]) => {
      const ratioSum = (rule.questions_ratio || 0) + (rule.reflection_ratio || 0) +
                       (rule.advice_ratio || 0) + (rule.silence_ratio || 0);
      if (Math.abs(ratioSum - 1.0) > 0.01) {
        errors.push(`Response rule "${modeName}" ratios sum to ${ratioSum}, must equal 1.0`);
      }
      if (!rule.max_length || rule.max_length <= 0) {
        errors.push(`Response rule "${modeName}" max_length must be a positive number`);
      }
    });
  } else {
    errors.push('Missing or invalid response_rules section');
  }

  // Check validation_gates
  if (config.validation_gates && Array.isArray(config.validation_gates)) {
    config.validation_gates.forEach((gate, idx) => {
      if (!gate.name) errors.push(`Validation gate ${idx} missing name`);
      if (!gate.gate_type) errors.push(`Validation gate ${idx} missing gate_type`);
      else if (!VALID_GATE_TYPES.includes(gate.gate_type)) {
        errors.push(`Validation gate ${idx} has invalid type "${gate.gate_type}". Valid types: ${VALID_GATE_TYPES.join(', ')}`);
      }
    });
  }

  // Check templates
  if (!config.templates || typeof config.templates !== 'object') {
    errors.push('Missing or invalid templates section');
  }

  return errors;
}

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

// POST /api/personas/:name/validate - Validate TOML and schema
app.post('/api/personas/:name/validate', (req, res) => {
  try {
    const { content } = req.body;

    // Parse TOML first
    let config;
    try {
      config = toml.parse(content);
    } catch (tomlError) {
      return res.status(400).json({
        success: false,
        error: `Invalid TOML: ${tomlError.message}`,
        validTOML: false
      });
    }

    // Validate schema
    const schemaErrors = validatePersonaSchema(config);

    if (schemaErrors.length > 0) {
      return res.status(400).json({
        success: false,
        validTOML: true,
        schemaErrors,
        error: `Schema validation failed: ${schemaErrors[0]}`
      });
    }

    res.json({
      success: true,
      validTOML: true,
      schemaValid: true,
      message: 'Valid TOML and schema',
      intentCount: Object.keys(config.intents || {}).length,
      responseRuleCount: Object.keys(config.response_rules || {}).length,
      validationGateCount: (config.validation_gates || []).length
    });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// GET /api/status - Server status
app.get('/api/status', (req, res) => {
  const files = fs.readdirSync(personasDir);
  const personaFiles = files.filter(f => f.endsWith('.toml'));

  const personas = personaFiles.map(f => {
    try {
      const content = fs.readFileSync(path.join(personasDir, f), 'utf-8');
      const config = toml.parse(content);
      return {
        name: f.replace('.toml', ''),
        version: config.version,
        description: config.description || '',
        intentCount: Object.keys(config.intents || {}).length,
        isValid: validatePersonaSchema(config).length === 0
      };
    } catch (e) {
      return {
        name: f.replace('.toml', ''),
        isValid: false,
        error: e.message
      };
    }
  });

  res.json({
    success: true,
    status: 'running',
    personasDir,
    personasCount: personaFiles.length,
    personas,
    schema: {
      validModes: VALID_MODES,
      validGateTypes: VALID_GATE_TYPES
    }
  });
});

// GET /api/schema - Get persona schema information
app.get('/api/schema', (req, res) => {
  res.json({
    success: true,
    schema: {
      validModes: VALID_MODES,
      validGateTypes: VALID_GATE_TYPES,
      requiredFields: ['name', 'version', 'capabilities', 'intents', 'response_rules', 'templates'],
      description: 'Generic Persona Configuration Schema for Indieclaw',
      documentation: 'See config/personas/README.md for detailed schema documentation'
    }
  });
});

// GET /api/personas/:name/metadata - Get persona metadata (name, version, description, counts)
app.get('/api/personas/:name/metadata', (req, res) => {
  try {
    const filePath = path.join(personasDir, `${req.params.name}.toml`);
    if (!fs.existsSync(filePath)) {
      return res.status(404).json({ success: false, error: 'Persona not found' });
    }

    const content = fs.readFileSync(filePath, 'utf-8');
    const config = toml.parse(content);
    const schemaErrors = validatePersonaSchema(config);

    res.json({
      success: true,
      name: req.params.name,
      version: config.version,
      description: config.description || '',
      textModel: config.text_model || 'qwen2:7b',
      visionModel: config.vision_model || 'llava:7b',
      capabilities: {
        canHandle: config.capabilities?.can_handle || [],
        cannotHandle: config.capabilities?.cannot_handle || [],
        requiresSearch: config.capabilities?.requires_search || [],
        internalOnly: config.capabilities?.internal_only || []
      },
      intents: Object.keys(config.intents || {}),
      responseModes: Object.keys(config.response_rules || {}),
      validationGateCount: (config.validation_gates || []).length,
      templateCount: Object.keys(config.templates || {}).length,
      isValid: schemaErrors.length === 0,
      schemaErrors: schemaErrors.length > 0 ? schemaErrors : undefined
    });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// ============ STRUCTURED PERSONA CONFIG ENDPOINTS ============

// GET /api/persona - Get structured persona config (form-friendly)
app.get('/api/persona', (req, res) => {
  try {
    const personaName = req.query.name || 'executive_coach';
    const filePath = path.join(personasDir, `${personaName}.toml`);

    console.log(`[/api/persona] Loading: ${personaName}`);
    console.log(`[/api/persona] Path: ${filePath}`);
    console.log(`[/api/persona] Exists: ${fs.existsSync(filePath)}`);

    if (!fs.existsSync(filePath)) {
      console.error(`[/api/persona] File not found at ${filePath}`);
      return res.status(404).json({ error: `Persona file not found at ${filePath}` });
    }

    const content = fs.readFileSync(filePath, 'utf8');
    const config = toml.parse(content);

    console.log(`[/api/persona] Successfully loaded ${personaName}`);
    res.json({
      name: personaName,
      config: config
    });
  } catch (err) {
    console.error(`[/api/persona] Error:`, err);
    res.status(500).json({ error: err.message });
  }
});

// POST /api/persona - Save structured persona config (form-friendly)
app.post('/api/persona', (req, res) => {
  try {
    const { config } = req.body;
    const personaName = req.query.name || 'executive_coach';

    console.log(`[POST /api/persona] Saving: ${personaName}`);

    if (!config) {
      return res.status(400).json({ error: 'No config provided' });
    }

    const filePath = path.join(personasDir, `${personaName}.toml`);
    const tomlString = tomlStringify(config);
    fs.writeFileSync(filePath, tomlString, 'utf8');

    console.log(`[POST /api/persona] Successfully saved ${personaName}`);
    res.json({
      success: true,
      message: `Persona '${personaName}' updated successfully`,
      path: filePath
    });
  } catch (err) {
    console.error(`[POST /api/persona] Error:`, err);
    res.status(500).json({ error: err.message });
  }
});

// ============ Server Start ============

app.listen(PORT, () => {
  console.log(`\n📝 Persona Control Plane is running!`);
  console.log(`🌐 Open your browser: http://localhost:${PORT}`);
  console.log(`📂 Managing personas in: ${personasDir}\n`);
  console.log(`📚 Available Endpoints:`);
  console.log(`   GET  /api/personas              - List all personas`);
  console.log(`   GET  /api/personas/:name        - Get persona TOML`);
  console.log(`   GET  /api/personas/:name/metadata - Get persona info`);
  console.log(`   POST /api/personas/:name        - Create/update persona`);
  console.log(`   POST /api/personas/:name/validate - Validate persona`);
  console.log(`   DELETE /api/personas/:name      - Delete persona`);
  console.log(`   GET  /api/status                - Server status & all personas`);
  console.log(`   GET  /api/schema                - Get schema information\n`);
});
