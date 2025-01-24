package rogue

import (
	"time"

	"github.com/wowsims/sod/sim/core"
)

func (rogue *Rogue) registerHemorrhageSpell() {
	if !rogue.Talents.Hemorrhage {
		return
	}

	spellID := map[int32]int32{
		40: 16511,
		50: 17347,
		60: 17348,
	}[rogue.Level]

	actionID := core.ActionID{SpellID: spellID}

	var numPlayers int
	for _, u := range rogue.Env.Raid.AllUnits {
		if u.Type == core.PlayerUnit {
			numPlayers++
		}
	}

	var hemoAuras core.AuraArray

	if numPlayers >= 2 {
		hemoAuras = rogue.NewEnemyAuraArray(func(target *core.Unit, level int32) *core.Aura {
			return core.HemorrhageAura(target, rogue.Level)
		})
	}

	rogue.Hemorrhage = rogue.RegisterSpell(core.SpellConfig{
		ClassSpellMask: ClassSpellMask_RogueHemorrhage,
		ActionID:       actionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          rogue.builderFlags(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   35.0,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		CritDamageBonus: rogue.lethality(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower())

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				if len(hemoAuras) > 0 {
					hemoAura := hemoAuras.Get(target)
					hemoAura.Activate(sim)
					hemoAura.SetStacks(sim, 30)
				}
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
