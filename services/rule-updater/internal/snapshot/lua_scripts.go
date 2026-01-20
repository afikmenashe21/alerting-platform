// Package snapshot handles building and writing rule snapshots to Redis.
package snapshot

import "github.com/redis/go-redis/v9"

// Lua scripts for direct Redis updates

const (
	// addRuleScript adds or updates a rule in the snapshot JSON directly in Redis
	addRuleScript = `
		local snapshot_key = KEYS[1]
		local version_key = KEYS[2]
		local rule_id = ARGV[1]
		local client_id = ARGV[2]
		local severity = ARGV[3]
		local source = ARGV[4]
		local name = ARGV[5]
		
		-- Load current snapshot
		local snapshot_json = redis.call('GET', snapshot_key)
		if not snapshot_json then
			-- Create empty snapshot
			snapshot_json = '{"schema_version":1,"severity_dict":{},"source_dict":{},"name_dict":{},"by_severity":{},"by_source":{},"by_name":{},"rules":{}}'
		end
		
		local snapshot = cjson.decode(snapshot_json)
		
		-- Find existing ruleInt for this rule_id
		local existing_rule_int = nil
		for rule_int_key, rule_info in pairs(snapshot.rules) do
			if rule_info.rule_id == rule_id then
				existing_rule_int = tonumber(rule_int_key)
				break
			end
		end
		
		-- Determine rule_int to use
		local rule_int
		if existing_rule_int then
			rule_int = existing_rule_int
			-- Remove from indexes first
			for sev, rule_ints in pairs(snapshot.by_severity) do
				for i = #rule_ints, 1, -1 do
					if rule_ints[i] == existing_rule_int then
						table.remove(rule_ints, i)
					end
				end
				if #rule_ints == 0 then
					snapshot.by_severity[sev] = nil
				end
			end
			for src, rule_ints in pairs(snapshot.by_source) do
				for i = #rule_ints, 1, -1 do
					if rule_ints[i] == existing_rule_int then
						table.remove(rule_ints, i)
					end
				end
				if #rule_ints == 0 then
					snapshot.by_source[src] = nil
				end
			end
			for nm, rule_ints in pairs(snapshot.by_name) do
				for i = #rule_ints, 1, -1 do
					if rule_ints[i] == existing_rule_int then
						table.remove(rule_ints, i)
					end
				end
				if #rule_ints == 0 then
					snapshot.by_name[nm] = nil
				end
			end
		else
			-- Find next available rule_int
			local max_rule_int = 0
			for rule_int_key, _ in pairs(snapshot.rules) do
				local rint = tonumber(rule_int_key)
				if rint and rint > max_rule_int then
					max_rule_int = rint
				end
			end
			rule_int = max_rule_int + 1
		end
		
		-- Add to dictionaries if needed
		if not snapshot.severity_dict[severity] then
			local max_sev = 0
			for _, v in pairs(snapshot.severity_dict) do
				if v > max_sev then max_sev = v end
			end
			snapshot.severity_dict[severity] = max_sev + 1
		end
		if not snapshot.source_dict[source] then
			local max_src = 0
			for _, v in pairs(snapshot.source_dict) do
				if v > max_src then max_src = v end
			end
			snapshot.source_dict[source] = max_src + 1
		end
		if not snapshot.name_dict[name] then
			local max_name = 0
			for _, v in pairs(snapshot.name_dict) do
				if v > max_name then max_name = v end
			end
			snapshot.name_dict[name] = max_name + 1
		end
		
		-- Add to indexes
		if not snapshot.by_severity[severity] then
			snapshot.by_severity[severity] = {}
		end
		table.insert(snapshot.by_severity[severity], rule_int)
		
		if not snapshot.by_source[source] then
			snapshot.by_source[source] = {}
		end
		table.insert(snapshot.by_source[source], rule_int)
		
		if not snapshot.by_name[name] then
			snapshot.by_name[name] = {}
		end
		table.insert(snapshot.by_name[name], rule_int)
		
		-- Add to rules map
		snapshot.rules[tostring(rule_int)] = {
			rule_id = rule_id,
			client_id = client_id
		}
		
		-- Write back and increment version
		local updated_json = cjson.encode(snapshot)
		redis.call('SET', snapshot_key, updated_json)
		return redis.call('INCR', version_key)
	`

	// removeRuleScript removes a rule from the snapshot JSON directly in Redis
	removeRuleScript = `
		local snapshot_key = KEYS[1]
		local version_key = KEYS[2]
		local rule_id = ARGV[1]
		
		-- Load current snapshot
		local snapshot_json = redis.call('GET', snapshot_key)
		if not snapshot_json then
			return 0
		end
		
		local snapshot = cjson.decode(snapshot_json)
		
		-- Find ruleInt for this rule_id
		local rule_int = nil
		for rule_int_key, rule_info in pairs(snapshot.rules) do
			if rule_info.rule_id == rule_id then
				rule_int = tonumber(rule_int_key)
				break
			end
		end
		
		if not rule_int then
			return 0
		end
		
		-- Remove from indexes
		for sev, rule_ints in pairs(snapshot.by_severity) do
			for i = #rule_ints, 1, -1 do
				if rule_ints[i] == rule_int then
					table.remove(rule_ints, i)
				end
			end
			if #rule_ints == 0 then
				snapshot.by_severity[sev] = nil
			end
		end
		for src, rule_ints in pairs(snapshot.by_source) do
			for i = #rule_ints, 1, -1 do
				if rule_ints[i] == rule_int then
					table.remove(rule_ints, i)
				end
			end
			if #rule_ints == 0 then
				snapshot.by_source[src] = nil
			end
		end
		for nm, rule_ints in pairs(snapshot.by_name) do
			for i = #rule_ints, 1, -1 do
				if rule_ints[i] == rule_int then
					table.remove(rule_ints, i)
				end
			end
			if #rule_ints == 0 then
				snapshot.by_name[nm] = nil
			end
		end
		
		-- Remove from rules map
		snapshot.rules[tostring(rule_int)] = nil
		
		-- Write back and increment version
		local updated_json = cjson.encode(snapshot)
		redis.call('SET', snapshot_key, updated_json)
		return redis.call('INCR', version_key)
	`
)

// newLuaScripts initializes the Lua scripts for the Writer.
func newLuaScripts() (*redis.Script, *redis.Script) {
	return redis.NewScript(addRuleScript), redis.NewScript(removeRuleScript)
}
