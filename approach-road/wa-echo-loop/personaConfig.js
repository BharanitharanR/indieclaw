const fs = require('fs');
const path = require('path');
const toml = require('toml');

class PersonaConfig {
    constructor(data) {
        this.name = data.persona?.name || 'Default Assistant';
        this.version = data.persona?.version || '1.0';
        this.textModel = data.models?.text_model || 'qwen3:8b';
        this.visionModel = data.models?.vision_model || 'gemma4:e2b';
        this.promptTemplate = data.personality?.prompt_template || '';
        this.tone = data.personality?.tone || 'professional';
        this.maxResponseLength = data.personality?.max_response_length || 1000;
        this.responseStyle = data.personality?.response_style || 'detailed';
        this.includeFollowup = data.personality?.include_followup_questions || false;
        this.toneGuidelines = data.personality?.tone_guidelines || {};
        this.allowedPhoneNumbers = data.whatsapp?.allowed_phone_numbers || [];
        this.enabled = data.whatsapp?.enabled !== false;
    }

    static loadByName(personaName = 'default') {
        const possiblePaths = [
            path.join(__dirname, 'config', 'personas', `${personaName}.toml`),
            path.join(__dirname, '../../gateway-service/config/personas', `${personaName}.toml`),
            path.join(__dirname, '../../../gateway-service/config/personas', `${personaName}.toml`),
        ];

        for (const filePath of possiblePaths) {
            if (fs.existsSync(filePath)) {
                console.log(`📂 Loading persona from: ${filePath}`);
                const content = fs.readFileSync(filePath, 'utf-8');
                const data = toml.parse(content);
                return new PersonaConfig(data);
            }
        }

        console.error(`❌ Persona file not found for '${personaName}' (tried: ${possiblePaths.join(', ')})`);
        // Return default persona as fallback
        return new PersonaConfig({
            persona: { name: 'Default Assistant', version: '1.0' },
            models: { text_model: 'qwen3:8b', vision_model: 'gemma4:e2b' },
            personality: {},
            whatsapp: { allowed_phone_numbers: [], enabled: true },
        });
    }

    getModelForType(hasMedia) {
        return hasMedia ? this.visionModel : this.textModel;
    }

    isPhoneAllowed(phoneNumber) {
        if (!this.enabled) {
            console.log('⚠️  WhatsApp is disabled in persona');
            return false;
        }

        if (this.allowedPhoneNumbers.length === 0) {
            console.log('⚠️  No phone numbers configured in persona');
            return false;
        }

        const normalized = this.normalizePhoneNumber(phoneNumber);
        const isAllowed = this.allowedPhoneNumbers.some(
            allowed => this.normalizePhoneNumber(allowed) === normalized
        );

        if (isAllowed) {
            console.log(`✅ Phone number ${phoneNumber} is allowed`);
        } else {
            console.log(`❌ Phone number ${phoneNumber} is NOT allowed`);
        }

        return isAllowed;
    }

    normalizePhoneNumber(number) {
        return number
            .toLowerCase()
            .replace(/-/g, '')
            .replace(/ /g, '')
            .replace(/\(/g, '')
            .replace(/\)/g, '')
            .replace(/\+/g, '')
            // Remove @c.us, @s.whatsapp.net suffixes if present
            .split('@')[0];
    }

    getAllowedPhoneNumbers() {
        return this.allowedPhoneNumbers;
    }

    toString() {
        return `Persona: ${this.name} (v${this.version}) | Tone: ${this.tone} | Allowed Numbers: ${this.allowedPhoneNumbers.length}`;
    }
}

module.exports = PersonaConfig;
