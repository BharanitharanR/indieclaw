// ============= STATE =============
let currentConfig = null;
let phoneNumbers = [];
const personaName = new URLSearchParams(window.location.search).get('persona') || 'executive_coach';

// ============= INITIALIZATION =============
document.addEventListener('DOMContentLoaded', () => {
    loadPersonasList();
    loadPersona();
    document.getElementById('personaForm').addEventListener('submit', savePersona);
});

// ============= LOAD PERSONAS LIST =============
async function loadPersonasList() {
    try {
        const response = await fetch('/api/personas');
        if (!response.ok) {
            throw new Error('Failed to load personas list');
        }

        const data = await response.json();
        const personas = data.personas || [];
        renderPersonasList(personas);
    } catch (err) {
        console.error('Error loading personas:', err);
        showMessage(`❌ Error loading personas: ${err.message}`, 'error');
    }
}

function renderPersonasList(personas) {
    const list = document.getElementById('personasList');
    list.innerHTML = '';

    const icons = {
        'executive_coach': '🎯',
        'default': '🤖',
        'tech_advisor': '💻',
        'health_coach': '🏋️',
        'life_coach': '✨',
    };

    personas.forEach(name => {
        const item = document.createElement('div');
        item.className = `persona-item ${name === personaName ? 'active' : ''}`;
        item.onclick = () => switchPersona(name);

        const icon = icons[name] || '👤';
        item.innerHTML = `<span class="persona-item-icon">${icon}</span>${name}`;
        list.appendChild(item);
    });
}

function switchPersona(newPersonaName) {
    // Update URL parameter
    const url = new URL(window.location);
    url.searchParams.set('persona', newPersonaName);
    window.history.pushState({}, '', url);

    // Reload with new persona
    personaName = newPersonaName;
    loadPersona();

    // Update active state in sidebar
    document.querySelectorAll('.persona-item').forEach(item => {
        item.classList.remove('active');
    });
    event.target.closest('.persona-item').classList.add('active');
}

function createNewPersona() {
    const newName = prompt('Enter new persona name (e.g., tech_advisor):');
    if (!newName) return;

    const cleanName = newName.toLowerCase().replace(/\s+/g, '_');

    // Create new persona with PersonaDefinition format (TOML)
    const newToml = `name = "${cleanName}"
version = 1
description = "A new persona"
text_model = "qwen2:7b"
vision_model = "llava:7b"

[capabilities]
can_handle = ["general questions"]
cannot_handle = []

[intents]
[intents.general_question]
mode = "research"
depth = "detailed"
confidence_min = 0.3

[response_rules]
[response_rules.research]
questions_ratio = 0.3
reflection_ratio = 0.3
advice_ratio = 0.2
silence_ratio = 0.2
max_length = 1500
forbidden_patterns = []
required_elements = []

[templates]
default = "You are a helpful assistant."

[[validation_gates]]
name = "length_check"
gate_type = "length_check"
[validation_gates.parameters]
max_length = 2000
`;

    // Save new persona using TOML endpoint
    fetch(`/api/personas/${cleanName}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: newToml })
    })
    .then(res => res.json())
    .then(data => {
        if (data.error) throw new Error(data.error);
        showMessage(`✅ Persona "${cleanName}" created`, 'success', 2000);
        loadPersonasList();
        switchPersona(cleanName);
    })
    .catch(err => showMessage(`❌ Error: ${err.message}`, 'error'));
}

// ============= LOAD PERSONA =============
async function loadPersona() {
    try {
        // Load full TOML content first
        const metaResponse = await fetch(`/api/personas/${personaName}`);
        if (!metaResponse.ok) {
            throw new Error(`Failed to load persona: ${metaResponse.statusText}`);
        }

        const metaData = await metaResponse.json();
        const tomlContent = metaData.content;

        // Parse TOML to get structured data
        const response = await fetch(`/api/personas/${personaName}/metadata`);
        if (!response.ok) {
            throw new Error(`Failed to load persona metadata: ${response.statusText}`);
        }

        const data = await response.json();
        currentConfig = {
            name: data.name,
            version: data.version,
            description: data.description,
            textModel: data.textModel,
            visionModel: data.visionModel,
            capabilities: data.capabilities,
            intents: data.intents,
            responseModes: data.responseModes,
            validationGates: data.validationGateCount,
            templates: data.templateCount,
            tomlContent: tomlContent,
            isValid: data.isValid,
            schemaErrors: data.schemaErrors
        };

        // Populate form fields with new structure
        populateForm();
        if (data.schemaErrors && data.schemaErrors.length > 0) {
            showMessage(`⚠️ Persona has schema errors`, 'warning', 3000);
        } else {
            showMessage('✅ Persona loaded successfully', 'success', 2000);
        }
    } catch (err) {
        showMessage(`❌ Error loading persona: ${err.message}`, 'error');
    }
}

// ============= POPULATE FORM =============
function populateForm() {
    if (!currentConfig) return;

    // Basic Info (New PersonaDefinition format)
    document.getElementById('personaName').value = currentConfig.name || '';
    document.getElementById('personaVersion').value = currentConfig.version || '';

    // Models
    document.getElementById('textModel').value = currentConfig.textModel || 'qwen2:7b';
    document.getElementById('visionModel').value = currentConfig.visionModel || 'llava:7b';

    // Show persona structure info
    const infoEl = document.getElementById('personaInfo');
    if (infoEl) {
        infoEl.innerHTML = `
            <div class="persona-info-box">
                <h3>Persona Structure</h3>
                <p><strong>Description:</strong> ${currentConfig.description || 'No description'}</p>
                <p><strong>Capabilities:</strong></p>
                <ul>
                    <li>Can Handle: ${currentConfig.capabilities?.canHandle?.join(', ') || 'none'}</li>
                    <li>Cannot Handle: ${currentConfig.capabilities?.cannotHandle?.join(', ') || 'none'}</li>
                </ul>
                <p><strong>Intents (${currentConfig.intents?.length || 0}):</strong> ${currentConfig.intents?.join(', ') || 'none'}</p>
                <p><strong>Response Modes (${currentConfig.responseModes?.length || 0}):</strong> ${currentConfig.responseModes?.join(', ') || 'none'}</p>
                <p><strong>Validation Gates:</strong> ${currentConfig.validationGates || 0}</p>
                <p><strong>Templates:</strong> ${currentConfig.templates || 0}</p>
                ${!currentConfig.isValid ? `<p style="color: red;">⚠️ Schema validation failed</p>` : '<p style="color: green;">✅ Valid schema</p>'}
            </div>
        `;
    }

    // Show raw TOML for editing
    if (currentConfig.tomlContent) {
        const tomlEditor = document.getElementById('tomlEditor');
        if (tomlEditor) {
            tomlEditor.value = currentConfig.tomlContent;
        }
    }

    // Render template list
    renderIntentsList();
}

// ============= RENDER INTENTS & MODES =============
function renderIntentsList() {
    const container = document.getElementById('intentsContainer');
    if (!container || !currentConfig.intents) return;

    container.innerHTML = '<h4>Intents Defined:</h4>';
    const ul = document.createElement('ul');
    currentConfig.intents.forEach(intent => {
        const li = document.createElement('li');
        li.textContent = intent;
        ul.appendChild(li);
    });
    container.appendChild(ul);

    const responsesContainer = document.getElementById('responsesContainer');
    if (responsesContainer && currentConfig.responseModes) {
        responsesContainer.innerHTML = '<h4>Response Modes:</h4>';
        const ul = document.createElement('ul');
        currentConfig.responseModes.forEach(mode => {
            const li = document.createElement('li');
            li.textContent = mode;
            ul.appendChild(li);
        });
        responsesContainer.appendChild(ul);
    }
}

// ============= PHONE NUMBERS =============
function renderPhoneNumbers() {
    const list = document.getElementById('phoneList');
    list.innerHTML = '';

    if (phoneNumbers.length === 0) {
        list.innerHTML = '<p style="color: #999; padding: 12px; text-align: center;">No phone numbers added yet</p>';
        return;
    }

    phoneNumbers.forEach((phone, index) => {
        const div = document.createElement('div');
        div.className = 'phone-item';
        div.innerHTML = `
            <span>${phone}</span>
            <button type="button" onclick="removePhoneNumber(${index})">Remove</button>
        `;
        list.appendChild(div);
    });
}

function addPhoneNumber() {
    const input = document.getElementById('newPhoneNumber');
    const phone = input.value.trim();

    if (!phone) {
        showMessage('❌ Please enter a phone number', 'error', 2000);
        return;
    }

    if (phoneNumbers.includes(phone)) {
        showMessage('❌ This phone number is already added', 'error', 2000);
        return;
    }

    phoneNumbers.push(phone);
    input.value = '';
    renderPhoneNumbers();
    showMessage('✅ Phone number added', 'success', 1500);
}

function removePhoneNumber(index) {
    phoneNumbers.splice(index, 1);
    renderPhoneNumbers();
    showMessage('✅ Phone number removed', 'success', 1500);
}

// ============= SAVE PERSONA =============
async function savePersona(e) {
    e.preventDefault();

    const saveBtn = document.getElementById('saveBtn');
    saveBtn.disabled = true;
    saveBtn.innerHTML = '<span class="loading"></span>Saving...';

    try {
        // Get TOML content from editor
        const tomlEditor = document.getElementById('tomlEditor');
        const tomlContent = tomlEditor?.value || currentConfig.tomlContent;

        if (!tomlContent || tomlContent.trim() === '') {
            throw new Error('No TOML content to save');
        }

        // Send to server using raw TOML endpoint
        const response = await fetch(`/api/personas/${personaName}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ content: tomlContent }),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save persona');
        }

        // Validate the persona after saving
        const validateResponse = await fetch(`/api/personas/${personaName}/validate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ content: tomlContent }),
        });

        const validateData = await validateResponse.json();
        if (!validateData.success) {
            showMessage(`⚠️ Saved but validation failed: ${validateData.error}`, 'warning', 4000);
        } else {
            showMessage(`✅ Persona saved and validated!`, 'success', 3000);
        }

        // Reload to show updated data
        setTimeout(() => loadPersona(), 500);
    } catch (err) {
        showMessage(`❌ Error: ${err.message}`, 'error');
    } finally {
        saveBtn.disabled = false;
        saveBtn.innerHTML = 'Save Changes';
    }
}

// ============= RESET FORM =============
function resetForm() {
    if (confirm('Are you sure? This will reload the current saved configuration.')) {
        loadPersona();
    }
}

// ============= MESSAGE DISPLAY =============
function showMessage(text, type, duration) {
    const messageEl = document.getElementById('message');
    messageEl.textContent = text;
    messageEl.className = `message show ${type}`;

    if (duration) {
        setTimeout(() => {
            messageEl.classList.remove('show');
        }, duration);
    }
}
