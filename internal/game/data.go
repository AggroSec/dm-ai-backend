package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Skill represents a single ability definition loaded from a JSON data file.
type Skill struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	LevelRequired int    `json:"level_required"`
	APCost        int    `json:"ap_cost"`
	WPCost        int    `json:"wp_cost"`
	HPCost        int    `json:"hp_cost"`
	Description   string `json:"description"`
	AIInstruction string `json:"ai_instruction"`
}

// ClassData represents a loaded class branch file.
type ClassData struct {
	Class       string  `json:"class"`
	Branch      string  `json:"branch"`
	Description string  `json:"description"`
	Skills      []Skill `json:"skills"`
}

// TalentInvestment tracks how many points a character has invested in a branch.
// Stored as JSONB in the characters table talents_invested column.
type TalentInvestment struct {
	Branch string `json:"branch"` // e.g. "base", "lightning", "hexblade"
	Points int    `json:"points"` // number of talent points invested
}

// Fate represents a single fate definition loaded from a JSON data file.
type Fate struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`   // driving or binding
	Domain        string `json:"domain"` // physical, magical, hybrid
	Description   string `json:"description"`
	Effect        string `json:"effect"`
	AIInstruction string `json:"ai_instruction"`
}

// LoadClassBranch loads a single class branch JSON file.
func LoadClassBranch(dataDir, class, branch string) (ClassData, error) {
	path := filepath.Join(dataDir, "classes", class, branch+".json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return ClassData{}, fmt.Errorf("failed to read class data %s/%s: %w", class, branch, err)
	}
	var data ClassData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return ClassData{}, fmt.Errorf("failed to parse class data %s/%s: %w", class, branch, err)
	}
	return data, nil
}

// LoadSkillsForCharacter loads all skills available to a character based on
// their class and actual talent investments. Only returns skills where
// level_required <= points invested in that branch.
// Base class skills (branch "base") are always loaded — level_required maps
// to the skill's position in the base progression.
func LoadSkillsForCharacter(dataDir, class string, talents []TalentInvestment) ([]Skill, error) {
	var available []Skill

	for _, talent := range talents {
		data, err := LoadClassBranch(dataDir, class, talent.Branch)
		if err != nil {
			// Skip missing branch files gracefully — don't fail the whole load
			continue
		}
		for _, skill := range data.Skills {
			if skill.LevelRequired <= talent.Points {
				available = append(available, skill)
			}
		}
	}

	return available, nil
}

// LoadFate loads a single fate definition by domain and ID.
func LoadFate(dataDir, domain, fateID string) (Fate, error) {
	path := filepath.Join(dataDir, "fates", domain, fateID+".json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Fate{}, fmt.Errorf("failed to read fate %s/%s: %w", domain, fateID, err)
	}
	var fate Fate
	if err := json.Unmarshal(bytes, &fate); err != nil {
		return Fate{}, fmt.Errorf("failed to parse fate %s/%s: %w", domain, fateID, err)
	}
	return fate, nil
}

// LoadAllFates loads every fate definition from all domain subdirectories.
// Used during character creation to present fate options.
func LoadAllFates(dataDir string) ([]Fate, error) {
	domains := []string{"physical", "magical", "hybrid"}
	var fates []Fate
	for _, domain := range domains {
		domainPath := filepath.Join(dataDir, "fates", domain)
		entries, err := os.ReadDir(domainPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read fates domain %s: %w", domain, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			bytes, err := os.ReadFile(filepath.Join(domainPath, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read fate file %s: %w", entry.Name(), err)
			}
			var fate Fate
			if err := json.Unmarshal(bytes, &fate); err != nil {
				return nil, fmt.Errorf("failed to parse fate file %s: %w", entry.Name(), err)
			}
			fates = append(fates, fate)
		}
	}
	return fates, nil
}

// LoadFatesForDomain loads all fates for a specific domain.
// Used during character creation when player has chosen a domain.
func LoadFatesForDomain(dataDir, domain string) ([]Fate, error) {
	domainPath := filepath.Join(dataDir, "fates", domain)
	entries, err := os.ReadDir(domainPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fates domain %s: %w", domain, err)
	}
	var fates []Fate
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		bytes, err := os.ReadFile(filepath.Join(domainPath, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read fate file %s: %w", entry.Name(), err)
		}
		var fate Fate
		if err := json.Unmarshal(bytes, &fate); err != nil {
			return nil, fmt.Errorf("failed to parse fate file %s: %w", entry.Name(), err)
		}
		fates = append(fates, fate)
	}
	return fates, nil
}
