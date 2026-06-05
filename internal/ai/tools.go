package ai

type Parameters struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Items       *Property           `json:"items,omitempty"` // For array types
	Properties  map[string]Property `json:"properties,omitempty"`
}

func GetToolDefinitions() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "end_combat",
				Description: "Used to end a combat before victory/defeat conditions are met. Used for special narrative circumstances.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"combat_id": {
							Type:        "string",
							Description: "The ID of the combat to end.",
						},
						"outcome": {
							Type:        "string",
							Description: "The outcome of the combat (e.g. 'retreat', 'surrender', 'narrative'). Can also give a small sentence so the players know why the combat ended.",
						},
					},
					Required: []string{"combat_id", "outcome"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "skip_turn",
				Description: "Used to skip a combatant's turn. Used for special narrative circumstances or if the combatant is incapacitated via status effects. You interpret the status effects but an example would be a status effect of paralyzed or stunned",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"combat_id": {
							Type:        "string",
							Description: "The ID of the combat.",
						},
						"entity": {
							Type:        "string",
							Description: "The id of the combatant whose turn it is.",
						},
						"reason": {
							Type:        "string",
							Description: "The reason for the turn skip. This is just for the players to understand why they lost their turn.",
						},
					},
					Required: []string{"combat_id", "entity"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "apply_damage",
				Description: "Used to apply HP damage or WP damage to a combatant. Used strictly for lowering one of these values. In combat, you call this specifically after getting the results of any rolls, modifiers, etc. so you can calculate the damage amount and then call this function to apply it. You can also use this outside of combat for narrative purposes, but it's primarily designed for combat.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"entity": {
							Type:        "string",
							Description: "The id of the character/NPC to apply damage to.",
						},
						"amount": {
							Type:        "integer",
							Description: "The amount of damage to apply.",
						},
						"type": {
							Type:        "string",
							Description: "The type of damage to apply (e.g. 'hp', 'wp').",
						},
						"source": {
							Type:        "string",
							Description: "The source of the damage (e.g. 'goblin attack', 'fireball spell'). This is just for logging purposes to understand where the damage came from.",
						},
					},
					Required: []string{"entity", "amount", "type"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "request_roll",
				Description: "Used to request a dice roll for a specific action or ability.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"sides": {
							Type:        "integer",
							Description: "The number of sides on the die to roll.",
						},
						"count": {
							Type:        "integer",
							Description: "The number of dice to roll.",
						},
					},
					Required: []string{"sides", "count"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "validate_action",
				Description: "Used to validate a player's skill/ability before it is executed in combat. supply the cost and server will validate if the player has the resources to perform the action and return a boolean. This is used to prevent players from performing actions they don't have the resources for and to help guide them towards valid actions. Will also deduct the resources if the action is valid so you don't have to worry about that part. For the associated cost, if there is none present just put 0 for the cost and it will validate based on the action alone. (e.g. 0 HP, 0 WP, 2 AP is an ability that just costs AP). you will process also need to validate an NPCs actions through this. For NPC combatants, continue taking actions until AP reaches 0. Always end NPC turn with at least a basic attack if no other action is available. Turn advances automatically when AP hits 0 for NPCs through this tool call, however finish up the rolls and damage/effect applying, etc before moving to the next combatants turn.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"action": {
							Type:        "string",
							Description: "The action to validate.",
						},
						"hp_cost": {
							Type:        "integer",
							Description: "The HP cost of the action.",
						},
						"wp_cost": {
							Type:        "integer",
							Description: "The WP cost of the action.",
						},
						"ap_cost": {
							Type:        "integer",
							Description: "The AP cost of the action.",
						},
						"combatant_id": {
							Type:        "string",
							Description: "The ID of the combatant performing the action. This is used to validate the action against the correct combatant's resources.",
						},
						"combat_id": {
							Type:        "string",
							Description: "The ID of the combat this action is being performed in. This is used to validate the action against the correct combatant's resources.",
						},
					},
					Required: []string{"action", "combatant_id", "combat_id", "hp_cost", "wp_cost", "ap_cost"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "apply_status_effect",
				Description: "Used to apply a status effect to a combatant.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"combatant_id": {
							Type:        "string",
							Description: "The ID of the combatant to apply the status effect to.",
						},
						"combat_id": {
							Type:        "string",
							Description: "The ID of the combat this is being applied in. This is used to apply the status effect to the correct combatant in the correct combat. This is option so that this tool allows for narrative calls. For narrative purposes do not call this on an NPC, only call on an NPC during combat, and always supply this if there is a combat.",
						},
						"effect": {
							Type:        "string",
							Description: "The status effect to apply.",
						},
						"duration": {
							Type:        "integer",
							Description: "The duration of the status effect.",
						},
						"persists": {
							Type:        "boolean",
							Description: "Whether the status effect persists outside of combat. This is for effects like poison that can last after combat ends or for narrative purposes.persistent effects will carry over to the next combat, non persistent effects will be removed after combat ends. Persistent effects will still expire and be removed if duration runs out, they just won't be automatically removed at the end of combat like non-persistent effects.",
						},
						"instruction": {
							Type:        "string",
							Description: "Set instructions here for yourself on what the status effect does, so you can recall when viewing the status effects for the combatant. For a player character use the description and instructions of the skill used, for NPCs if you make up a status effect leave yourself instructions.",
						},
					},
					Required: []string{"combatant_id", "effect", "duration", "persists", "instruction"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "start_combat",
				Description: "Used to start a new combat. returns the full combat session details.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"npcs": {
							Type:        "array",
							Description: "Array of NPC combatants entering combat. Each NPC should include name, hp, max_hp, and all six stats (strength, dexterity, fortitude, willpower, alacrity, wisdom).",
							Items: &Property{
								Type: "object",
							},
						},
					},
					Required: []string{"npcs"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_combat_state",
				Description: "Used to get the current state of an ongoing combat. returns the full combat session details.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"combat_id": {
							Type:        "string",
							Description: "The ID of the combat to get the state of.",
						},
					},
					Required: []string{"combat_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "action_failed",
				Description: "Used to indicate that a player's action has failed validation.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character whose action failed.",
						},
						"reason": {
							Type:        "string",
							Description: "The reason why the action failed.",
						},
					},
					Required: []string{"character_id", "reason"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "award_xp",
				Description: "Used to award experience points to a character.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character to award XP to.",
						},
						"amount": {
							Type:        "integer",
							Description: "The amount of XP to award.",
						},
					},
					Required: []string{"character_id", "amount"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "apply_heal",
				Description: "Used to increase HP or WP for a combatant. Used strictly for raising one of these values. In combat, you call this specifically after getting the results of any rolls, modifiers, etc. so you can calculate the heal amount and then call this function to apply it. You can also use this outside of combat for narrative purposes, but it's primarily designed for combat.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"entity": {
							Type:        "string",
							Description: "The id of the character/NPC to apply the heal to.",
						},
						"amount": {
							Type:        "integer",
							Description: "The amount of healing to apply.",
						},
						"type": {
							Type:        "string",
							Description: "The type of healing to apply (e.g. 'hp', 'wp').",
						},
						"source": {
							Type:        "string",
							Description: "The source of the healing (e.g. 'healing potion', 'medic skill'). This is just for logging purposes to understand where the healing came from.",
						},
					},
					Required: []string{"entity", "amount", "type"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_character",
				Description: "Used to retrieve information about a specific character. Returns the full character sheet details for the ID supplied.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character to retrieve.",
						},
					},
					Required: []string{"character_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_skills",
				Description: "Used to retrieve the skills of a specific character. Used if you need to reference or supply the player with options on what they have ability-wise. Returns an array of the characters skills with all details for each skill.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character whose skills to retrieve.",
						},
					},
					Required: []string{"character_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "remove_status_effect",
				Description: "Used to remove a status effect from a character. This is used for when an effect needs to be removed before the duration runs out, such as a cleanse or a status effect that is only supposed to last for one turn. A specific example from the current character classes is etch rune, where they detonate damage when a physical attack lands (or they use the skill the automatically detonates them. So you would apply the status effect with the instruction to detonate when a physical attack lands or when they use the skill again, and then you would remove the status effect after it detonates to prevent it from detonating again on the next physical attack or skill use.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character from whom to remove the status effect.",
						},
						"status_id": {
							Type:        "string",
							Description: "The ID of the status effect to remove.",
						},
					},
					Required: []string{"character_id", "status_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "equip_item",
				Description: "Used to equip an item on a character. This is used for when a character wants to use an item that they have in their inventory. If used during combat, validate the action first for a cost of 1 AP",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character to equip the item on.",
						},
						"item_id": {
							Type:        "string",
							Description: "The ID of the item to equip.",
						},
						"slot": {
							Type:        "string",
							Description: "The slot to equip the item in (e.g. 'head', 'body', 'legs', 'weapon'). This is used to determine where the item is equipped and what bonuses it provides.",
						},
					},
					Required: []string{"character_id", "item_id", "slot"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "rest",
				Description: "Used to rest and recover health and willpower. This is used for when a character wants to rest to recover HP and WP. cannot be used during combat, removes all status effects (good and bad). If they have something that is supposed to be a lifetime effect, reapply it after resting. short rest recovers a portion of HP and WP. Long rest fully recovers them both. Narratively time passes with either and effects time sensitive items, use your judgement on allowing a rest before calling this tool.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character to rest.",
						},
						"rest_type": {
							Type:        "string",
							Description: "The type of rest to take (e.g. 'short', 'long'). This is used to determine how much HP and WP the character recovers.",
						},
						"location": {
							Type:        "string",
							Description: "The location where the character is resting. This is just for narrative purposes to add flavor to the resting action.",
						},
					},
					Required: []string{"character_id", "rest_type"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "update_character",
				Description: "Used to update a character's information. Use almost exclusively for character creation which will be free form with the player as you flesh out their character with them.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character to update.",
						},
						"name": {
							Type:        "string",
							Description: "The name of the character.",
						},
						"class": {
							Type:        "string",
							Description: "The class of the character.",
						},
						"level": {
							Type:        "integer",
							Description: "The level of the character. Usually starts at 1, but if you decide you want to start them at a higher level(more experienced player) then you can do so. Preference is to start at 1 and use award_xp tool to the desired level so they can go through the level up process. you can update incrementally during character creation as well, all parameters are optional except the character ID",
						},
						"strength": {
							Type:        "integer",
							Description: "The strength stat of the character.",
						},
						"dexterity": {
							Type:        "integer",
							Description: "The dexterity stat of the character.",
						},
						"fortitude": {
							Type:        "integer",
							Description: "The fortitude stat of the character.",
						},
						"willpower": {
							Type:        "integer",
							Description: "The willpower stat of the character.",
						},
						"alacrity": {
							Type:        "integer",
							Description: "The alacrity stat of the character.",
						},
						"wisdom": {
							Type:        "integer",
							Description: "The wisdom stat of the character.",
						},
						"driving_fate": {
							Type:        "string",
							Description: "The driving fate stat of the character. see system prompt for details on driving and binding fates.",
						},
						"binding_fate": {
							Type:        "string",
							Description: "The binding fate stat of the character. see system prompt for details on driving and binding fates.",
						},
						"talents_invested": {
							Type:        "array",
							Description: "An array of the talents the character has invested in. Talent investments control the skills a character has access to. see system prompt for details on the talent system.",
							Items: &Property{
								Type: "object",
							},
						},
					},
					Required: []string{"character_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "give_item",
				Description: "Used to give an item to a character. You create all loot/items to fit the game system through this tool. This is used for when a character acquires a new item, whether through looting, purchasing, or as a quest reward.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Property{
						"character_id": {
							Type:        "string",
							Description: "The ID of the character to give the item to.",
						},
						"item": {
							Type:        "object",
							Description: "The item to give to the character. This should include all relevant details about the item, such as name, stats, etc. description is up to you narratively, but make sure to include all the relevant mechanical details in the item properties so the player can reference them when deciding what to do with the item.",
							Properties: map[string]Property{
								"name": {
									Type:        "string",
									Description: "The name of the item.",
								},
								"id": {
									Type:        "string",
									Description: "The ID of the item. create a unique ID for the item so it can be referenced later, especially if it's going to be equipped and provide bonuses or have an active ability that can be used in combat.",
								},
								"type": {
									Type:        "string",
									Description: "The type of the item (e.g. 'weapon', 'armor', 'consumable'). This is used to determine how the item can be used and what bonuses it provides.",
								},
								"armor_type": {
									Type:        "string",
									Description: "The type of armor (e.g. 'light', 'medium', 'heavy', 'none'). This is used to determine the armor's properties and how it affects the character.",
								},
								"base_defense": {
									Type:        "integer",
									Description: "The base defense provided by the item. This is used to calculate the character's total defense when the item is equipped.",
								},
								"ap_cost": {
									Type:        "integer",
									Description: "The action points cost to use the item.",
								},
								"strength": {
									Type:        "integer",
									Description: "The strength bonus provided by the item.",
								},
								"dexterity": {
									Type:        "integer",
									Description: "The dexterity bonus provided by the item.",
								},
								"fortitude": {
									Type:        "integer",
									Description: "The fortitude bonus provided by the item.",
								},
								"willpower": {
									Type:        "integer",
									Description: "The willpower bonus provided by the item.",
								},
								"alacrity": {
									Type:        "integer",
									Description: "The alacrity bonus provided by the item.",
								},
								"wisdom": {
									Type:        "integer",
									Description: "The wisdom bonus provided by the item.",
								},
							},
						},
					},
					Required: []string{"character_id", "item"},
				},
			},
		},
	}
}
