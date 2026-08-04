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

    // Create new persona config
    const newConfig = currentConfig ? JSON.parse(JSON.stringify(currentConfig)) : {
        persona: { name: cleanName, version: '1.0' },
        models: { text_model: 'qwen3:8b', vision_model: 'gemma4:e2b' },
        personality: {
            prompt_template: 'You are a helpful assistant.',
            planner_prompt: 'Evaluate if this question is within scope.',
            tone: 'friendly',
            max_response_length: 1500,
            response_style: 'conversational',
            include_followup_questions: true,
            tone_guidelines: { empathy: '', accountability: '', clarity: '', confidence: '', curiosity: '' }
        },
        whatsapp: { allowed_phone_numbers: [], enabled: true }
    };

    newConfig.persona.name = cleanName;

    // Save new persona using structured API
    fetch(`/api/persona?name=${cleanName}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config: newConfig })
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
        const response = await fetch(`/api/persona?name=${personaName}`);
        if (!response.ok) {
            throw new Error(`Failed to load persona: ${response.statusText}`);
        }

        const data = await response.json();
        currentConfig = data.config;
        phoneNumbers = data.config.whatsapp?.allowed_phone_numbers || [];

        // Populate form fields
        populateForm();
        showMessage('✅ Persona loaded successfully', 'success', 2000);
    } catch (err) {
        showMessage(`❌ Error loading persona: ${err.message}`, 'error');
    }
}

// ============= POPULATE FORM =============
function populateForm() {
    if (!currentConfig) return;

    // Basic Info
    document.getElementById('personaName').value = currentConfig.persona?.name || '';
    document.getElementById('personaVersion').value = currentConfig.persona?.version || '';

    // Models
    document.getElementById('textModel').value = currentConfig.models?.text_model || '';
    document.getElementById('visionModel').value = currentConfig.models?.vision_model || '';

    // Personality
    document.getElementById('promptTemplate').value = currentConfig.personality?.prompt_template || '';
    document.getElementById('plannerPrompt').value = currentConfig.personality?.planner_prompt || '';
    document.getElementById('tone').value = currentConfig.personality?.tone || '';
    document.getElementById('maxResponseLength').value = currentConfig.personality?.max_response_length || '';
    document.getElementById('responseStyle').value = currentConfig.personality?.response_style || '';
    document.getElementById('includeFollowup').checked = currentConfig.personality?.include_followup_questions === true;

    // Tone Guidelines
    const tg = currentConfig.personality?.tone_guidelines || {};
    document.getElementById('toneEmpathy').value = tg.empathy || '';
    document.getElementById('toneAccountability').value = tg.accountability || '';
    document.getElementById('toneClarity').value = tg.clarity || '';
    document.getElementById('toneConfidence').value = tg.confidence || '';
    document.getElementById('toneCuriosity').value = tg.curiosity || '';

    // WhatsApp
    document.getElementById('whatsappEnabled').checked = currentConfig.whatsapp?.enabled !== false;

    // Render phone numbers
    renderPhoneNumbers();
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
        // Build config object from form
        const config = {
            persona: {
                name: document.getElementById('personaName').value,
                version: document.getElementById('personaVersion').value,
            },
            models: {
                text_model: document.getElementById('textModel').value,
                vision_model: document.getElementById('visionModel').value,
            },
            personality: {
                prompt_template: document.getElementById('promptTemplate').value,
                planner_prompt: document.getElementById('plannerPrompt').value,
                tone: document.getElementById('tone').value,
                max_response_length: parseInt(document.getElementById('maxResponseLength').value),
                response_style: document.getElementById('responseStyle').value,
                include_followup_questions: document.getElementById('includeFollowup').checked,
                tone_guidelines: {
                    empathy: document.getElementById('toneEmpathy').value,
                    accountability: document.getElementById('toneAccountability').value,
                    clarity: document.getElementById('toneClarity').value,
                    confidence: document.getElementById('toneConfidence').value,
                    curiosity: document.getElementById('toneCuriosity').value,
                },
            },
            whatsapp: {
                allowed_phone_numbers: phoneNumbers,
                enabled: document.getElementById('whatsappEnabled').checked,
            },
        };

        // Validate required fields
        if (!config.persona.name || !config.models.text_model) {
            throw new Error('Please fill in all required fields');
        }

        // Send to server
        const response = await fetch(`/api/persona?name=${personaName}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ config }),
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save persona');
        }

        const result = await response.json();
        currentConfig = config;
        showMessage(`✅ ${result.message}`, 'success', 3000);
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
