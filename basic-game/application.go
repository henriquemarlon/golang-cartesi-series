package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rollmelette/rollmelette"
)

type GameState struct {
	Monsters map[string]Monster `json:"monsters"`
}

type Monster struct {
	Name      string `json:"name"`
	HitPoints int    `json:"hitPoints"`
}

type InputKind string

const (
	AddMonster    InputKind = "AddMonster"
	AttackMonster InputKind = "AttackMonster"
)

type Input struct {
	Kind    InputKind       `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type AddMonsterPayload = Monster

type AttackMonsterPayload struct {
	MonsterName string `json:"monsterName"`
	Damage      int    `json:"damage"`
}

type GameApplication struct {
	gm    common.Address
	state GameState
}

func NewGameApplication(gm common.Address) *GameApplication {
	return &GameApplication{
		gm: gm,
		state: GameState{
			Monsters: make(map[string]Monster),
		},
	}
}

func (a *GameApplication) Advance(
	env rollmelette.Env,
	metadata rollmelette.Metadata,
	deposit rollmelette.Deposit,
	payload []byte,
) error {
	var input Input
	err := json.Unmarshal(payload, &input)
	if err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	switch input.Kind {
	case AddMonster:
		var inputPayload AddMonsterPayload
		err = json.Unmarshal(input.Payload, &inputPayload)
		if err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		err = a.handleAddMonster(metadata, inputPayload)
		if err != nil {
			return err
		}
	case AttackMonster:
		var inputPayload AttackMonsterPayload
		err = json.Unmarshal(input.Payload, &inputPayload)
		if err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		err = a.handleAttackMonster(inputPayload)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid input kind: %v", input.Kind)
	}
	return a.Inspect(env, nil)
}

func (a *GameApplication) Inspect(env rollmelette.EnvInspector, payload []byte) error {
	bytes, err := json.Marshal(a.state)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	env.Report(bytes)
	return nil
}

func (a *GameApplication) handleAddMonster(
	metadata rollmelette.Metadata,
	inputPayload AddMonsterPayload,
) error {
	if metadata.MsgSender != a.gm {
		return fmt.Errorf("only GM can add monsters")
	}
	if inputPayload.HitPoints <= 0 {
		return fmt.Errorf("hit points must be positive")
	}
	_, ok := a.state.Monsters[inputPayload.Name]
	if ok {
		return fmt.Errorf("monster with this name already exists")
	}
	a.state.Monsters[inputPayload.Name] = inputPayload
	return nil
}

func (a *GameApplication) handleAttackMonster(inputPayload AttackMonsterPayload) error {
	if inputPayload.Damage < 0 {
		return fmt.Errorf("negative damage")
	}
	monster, ok := a.state.Monsters[inputPayload.MonsterName]
	if !ok {
		return fmt.Errorf("monster not found")
	}
	monster.HitPoints -= inputPayload.Damage
	if monster.HitPoints <= 0 {
		delete(a.state.Monsters, inputPayload.MonsterName)
	} else {
		a.state.Monsters[inputPayload.MonsterName] = monster
	}
	return nil
}

func main() {
	ctx := context.Background()
	opts := rollmelette.NewRunOpts()
	app := new(GameApplication)
	err := rollmelette.Run(ctx, opts, app)
	if err != nil {
		slog.Error("application error", "error", err)
	}
}
