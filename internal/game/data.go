package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Branch string `json:"branch"`
	Points int    `json:"points"`
}

// Fate represents a single fate definition loaded from a JSON data file.
type Fate struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Domain        string `json:"domain"`
	Description   string `json:"description"`
	Effect        string `json:"effect"`
	AIInstruction string `json:"ai_instruction"`
}

// SkillWithStatus represents a skill with its availability status for a character.
type SkillWithStatus struct {
	Skill
	Branch string `json:"branch"`
	Status string `json:"status"` // "unlocked", "available", "locked"
}

// BranchSkills groups skills by branch with their status.
type BranchSkills struct {
	Description string            `json:"description"`
	Skills      []SkillWithStatus `json:"skills"`
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
func LoadSkillsForCharacter(dataDir, class string, talents []TalentInvestment) ([]Skill, error) {
	var available []Skill

	for _, talent := range talents {
		data, err := LoadClassBranch(dataDir, class, talent.Branch)
		if err != nil {
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

// LoadAllSkillsForCharacter loads the full skill tree for a character's class
// across all branches, categorizing each skill as:
// - "unlocked": talent points invested meet level_required
// - "available": character has unspent talent points and investing would meet level_required
// - "locked": level_required not yet met even with available points
func LoadAllSkillsForCharacter(dataDir, class string, talents []TalentInvestment, talentPointsAvailable int) (map[string]BranchSkills, error) {
	investedPoints := make(map[string]int)
	for _, t := range talents {
		investedPoints[t.Branch] = t.Points
	}

	classPath := filepath.Join(dataDir, "classes", class)
	entries, err := os.ReadDir(classPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read class directory %s: %w", class, err)
	}

	result := make(map[string]BranchSkills)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		branchName := strings.TrimSuffix(entry.Name(), ".json")

		data, err := LoadClassBranch(dataDir, class, branchName)
		if err != nil {
			return nil, err
		}

		points := investedPoints[branchName]
		var skillsWithStatus []SkillWithStatus

		for _, skill := range data.Skills {
			var status string
			if skill.LevelRequired <= points {
				status = "unlocked"
			} else if skill.LevelRequired <= points+talentPointsAvailable {
				status = "available"
			} else {
				status = "locked"
			}
			skillsWithStatus = append(skillsWithStatus, SkillWithStatus{
				Skill:  skill,
				Branch: branchName,
				Status: status,
			})
		}

		result[branchName] = BranchSkills{
			Description: data.Description,
			Skills:      skillsWithStatus,
		}
	}

	return result, nil
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
