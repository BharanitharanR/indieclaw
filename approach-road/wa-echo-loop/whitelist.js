const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

/**
 * Whitelist Manager
 * Tracks registered contact names based on configured prefix
 * Supports both in-memory and file-based storage
 */
class WhitelistManager {
    constructor(config) {
        this.config = {
            enabled: config.enabled !== false,
            contactNamePrefix: config.contactNamePrefix || 'USER',
            storage: config.storage || 'memory',  // 'memory' or 'file'
            storagePath: config.storagePath || './whitelist.json',
        };

        // In-memory storage
        this.whitelist = new Set();

        // Initialize
        this.initialized = false;
        this.loadWhitelist();
    }

    /**
     * Load whitelist from storage
     */
    async loadWhitelist() {
        if (!this.config.enabled) {
            console.log('[Whitelist] ⚠️  Whitelist is disabled in persona config');
            this.initialized = true;
            return;
        }

        if (this.config.storage === 'file') {
            this.loadFromFile();
        } else {
            console.log('[Whitelist] 📝 Using in-memory storage');
        }
        this.initialized = true;
    }

    /**
     * Load whitelist from file
     */
    loadFromFile() {
        try {
            if (fs.existsSync(this.config.storagePath)) {
                const data = fs.readFileSync(this.config.storagePath, 'utf-8');
                const entries = JSON.parse(data);

                // Load into memory
                if (Array.isArray(entries)) {
                    entries.forEach(entry => {
                        if (entry.contact_name) {
                            this.whitelist.add(entry.contact_name);
                        }
                    });
                }
                console.log(`[Whitelist] ✅ Loaded ${this.whitelist.size} entries from file`);
            } else {
                console.log(`[Whitelist] 📄 No existing whitelist file, starting fresh`);
            }
        } catch (err) {
            console.error(`[Whitelist] ❌ Error loading whitelist: ${err.message}`);
        }
    }

    /**
     * Save whitelist to file
     */
    saveToFile() {
        if (this.config.storage !== 'file') return;

        try {
            const entries = Array.from(this.whitelist).map(contactName => ({
                contact_name: contactName,
                added_at: new Date().toISOString(),
            }));

            fs.writeFileSync(this.config.storagePath, JSON.stringify(entries, null, 2));
            console.log(`[Whitelist] 💾 Saved ${entries.length} entries to file`);
        } catch (err) {
            console.error(`[Whitelist] ❌ Error saving whitelist: ${err.message}`);
        }
    }

    /**
     * Check if a contact name is valid for registration
     * Must match the configured prefix format
     */
    isValidFormat(contactName) {
        if (!contactName) return false;

        // Format must start with configured prefix and have a separator
        const expectedPrefix = this.config.contactNamePrefix;
        return contactName.startsWith(`${expectedPrefix}-`);
    }

    /**
     * Add a contact name to the whitelist
     * Returns: { success: boolean, message: string }
     * Note: Accepts any contact name format (lenient for registration)
     */
    add(contactName) {
        if (!this.config.enabled) {
            return {
                success: false,
                message: '[Whitelist] Whitelist is disabled',
            };
        }

        if (!contactName || contactName.trim() === '') {
            return {
                success: false,
                message: '[Whitelist] ❌ Contact name cannot be empty',
            };
        }

        const trimmedName = contactName.trim();

        if (this.whitelist.has(trimmedName)) {
            return {
                success: false,
                message: `[Whitelist] ℹ️  Contact already whitelisted: ${trimmedName}`,
            };
        }

        // Check format validity (for logging only, not blocking)
        const isValidFormat = this.isValidFormat(trimmedName);
        if (!isValidFormat) {
            console.warn(`[Whitelist] ⚠️  Format mismatch: "${trimmedName}" (expected: ${this.config.contactNamePrefix}-*)`);
            console.log(`[Whitelist] ℹ️  Proceeding anyway - accepting contact name as-is`);
        }

        this.whitelist.add(trimmedName);
        this.saveToFile();

        console.log(`[Whitelist] ✅ Added ${trimmedName} to whitelist`);
        return {
            success: true,
            message: `[Whitelist] ✅ Contact whitelisted: ${trimmedName}`,
            contactName: trimmedName,
            formatWarning: !isValidFormat ? `Expected format: ${this.config.contactNamePrefix}-*` : null,
        };
    }

    /**
     * Check if a contact name is on the whitelist
     */
    isWhitelisted(contactName) {
        if (!this.config.enabled) {
            return false;
        }
        return this.whitelist.has(contactName);
    }

    /**
     * Remove a contact name from whitelist
     */
    remove(contactName) {
        if (!this.config.enabled) {
            return { success: false, message: '[Whitelist] Whitelist is disabled' };
        }

        if (!this.whitelist.has(contactName)) {
            return {
                success: false,
                message: `[Whitelist] Contact not found: ${contactName}`,
            };
        }

        this.whitelist.delete(contactName);
        this.saveToFile();

        console.log(`[Whitelist] ✅ Removed ${contactName} from whitelist`);
        return {
            success: true,
            message: `[Whitelist] ✅ Contact removed: ${contactName}`,
        };
    }

    /**
     * Get all whitelisted contact names
     */
    getAll() {
        return Array.from(this.whitelist);
    }

    /**
     * Get count of whitelisted contacts
     */
    getCount() {
        return this.whitelist.size;
    }

    /**
     * Clear all whitelisted contacts (admin use only)
     */
    clear() {
        const count = this.whitelist.size;
        this.whitelist.clear();
        this.saveToFile();
        console.log(`[Whitelist] ⚠️  Cleared ${count} entries`);
        return { success: true, count };
    }

    /**
     * Get whitelist status/info
     */
    getStatus() {
        return {
            enabled: this.config.enabled,
            initialized: this.initialized,
            contactNamePrefix: this.config.contactNamePrefix,
            storage: this.config.storage,
            storagePath: this.config.storagePath,
            count: this.whitelist.size,
            entries: this.getAll(),
        };
    }

    /**
     * Log a routing decision
     */
    logRoutingDecision(contactName, isWhitelisted, action) {
        const status = isWhitelisted ? '✅ REGISTERED' : '❌ UNREGISTERED';
        const route = action === 'orchestrator' ? '→ orchestrator.requests' : '→ adiyan.registration.inbound';
        console.log(`[Whitelist] [Routing] ${status} | ${contactName} | ${route}`);
    }
}

module.exports = WhitelistManager;
