const fs = require('fs');
const path = require('path');

class WhitelistManager {
    constructor(config = {}) {
        this.config = {
            enabled: config.enabled !== false,
            contactNamePrefix: config.contactNamePrefix || 'USER',
            storage: config.storage || 'memory',
            storagePath: config.storagePath || './whitelist.json',
        };

        this.whitelist = new Set();
        this.loadWhitelist();
    }

    loadWhitelist() {
        if (this.config.storage === 'file') {
            this.loadFromFile();
        }
    }

    loadFromFile() {
        try {
            const storagePath = path.join(__dirname, this.config.storagePath);
            if (fs.existsSync(storagePath)) {
                const data = JSON.parse(fs.readFileSync(storagePath, 'utf-8'));
                this.whitelist = new Set(data);
                console.log(`✅ Loaded ${this.whitelist.size} entries from whitelist file`);
            }
        } catch (err) {
            console.warn(`⚠️  Failed to load whitelist from file: ${err.message}`);
        }
    }

    saveToFile() {
        try {
            const storagePath = path.join(__dirname, this.config.storagePath);
            fs.writeFileSync(storagePath, JSON.stringify(Array.from(this.whitelist), null, 2));
        } catch (err) {
            console.warn(`⚠️  Failed to save whitelist to file: ${err.message}`);
        }
    }

    isValidFormat(contactName) {
        // Lenient mode: accept any contact name format
        if (!contactName || contactName.trim().length === 0) {
            return false;
        }

        const hasPrefix = contactName.toUpperCase().includes(this.config.contactNamePrefix.toUpperCase());
        if (!hasPrefix) {
            console.warn(`⚠️  Contact name "${contactName}" doesn't match prefix "${this.config.contactNamePrefix}". Still accepting (lenient mode).`);
        }

        return true;
    }

    add(contactName) {
        if (!this.isValidFormat(contactName)) {
            return { success: false, message: 'Invalid contact name format' };
        }

        this.whitelist.add(contactName);
        this.saveToFile();
        return { success: true, message: `Added ${contactName} to whitelist` };
    }

    isWhitelisted(contactName) {
        return this.whitelist.has(contactName);
    }

    remove(contactName) {
        if (!this.whitelist.has(contactName)) {
            return { success: false, message: `${contactName} not in whitelist` };
        }

        this.whitelist.delete(contactName);
        this.saveToFile();
        return { success: true, message: `Removed ${contactName} from whitelist` };
    }

    getAll() {
        return Array.from(this.whitelist);
    }

    getCount() {
        return this.whitelist.size;
    }

    getStatus() {
        return {
            enabled: this.config.enabled,
            count: this.whitelist.size,
            prefix: this.config.contactNamePrefix,
            storage: this.config.storage,
            entries: this.getAll()
        };
    }

    logRoutingDecision(contactName, isWhitelisted, action) {
        const status = isWhitelisted ? '✅' : '❌';
        console.log(`[Routing] ${status} ${contactName} → ${action}`);
    }
}

module.exports = WhitelistManager;
