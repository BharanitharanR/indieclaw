const fs = require('fs');
const path = require('path');
const toml = require('toml');

class PersonaConfig {
    constructor(data) {
        this.name = data.name || 'Default';
        this.version = data.version || '1.0';
        this.tone = data.tone || 'professional';
        this.systemPrompt = data.system_prompt || '';
        this.plannerPrompt = data.planner_prompt || '';

        this.whitelistConfig = {
            enabled: data.whitelist?.enabled !== false,
            contactNamePrefix: data.whitelist?.contact_name_prefix || 'USER',
            storage: data.whitelist?.storage || 'memory',
            storagePath: data.whitelist?.storage_path || './whitelist.json',
        };
    }

    toString() {
        return `Persona: ${this.name} (v${this.version}) | Tone: ${this.tone} | Whitelist: ${this.whitelistConfig.contactNamePrefix}-*`;
    }

    getWhitelistConfig() {
        return this.whitelistConfig;
    }

    getContactNamePrefix() {
        return this.whitelistConfig.contactNamePrefix;
    }

    isWhitelistEnabled() {
        return this.whitelistConfig.enabled;
    }

    static loadByName(name) {
        const configPath = path.join(__dirname, '../config/personas', `${name}.toml`);

        try {
            if (fs.existsSync(configPath)) {
                const data = toml.parse(fs.readFileSync(configPath, 'utf-8'));
                return new PersonaConfig(data);
            }
        } catch (err) {
            console.warn(`Could not load persona config for "${name}":`, err.message);
        }

        return new PersonaConfig({ name: 'Default Assistant' });
    }
}

module.exports = PersonaConfig;
