-- +goose Up
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TYPE combat_status AS ENUM ('active', 'ended');

CREATE TABLE combat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    status combat_status NOT NULL DEFAULT 'active',
    round INT NOT NULL DEFAULT 1,
    current_turn UUID NOT NULL,
    turn_order JSONB NOT NULL DEFAULT '[]',
    combatants JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE combat_sessions;
DROP TABLE campaigns;
DROP TYPE combat_status;