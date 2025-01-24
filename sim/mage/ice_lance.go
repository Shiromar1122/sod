package mage

import (
	"time"

	"github.com/wowsims/sod/sim/core"
	"github.com/wowsims/sod/sim/core/proto"
)

// TODO: Classic review ice lance numbers on live
func (mage *Mage) registerIceLanceSpell() {
	if !mage.HasRune(proto.MageRune_RuneHandsIceLance) {
		return
	}

	hasWintersChillTalent := mage.Talents.WintersChill > 0

	baseDamageLow := mage.baseRuneAbilityDamage() * 0.55
	baseDamageHigh := mage.baseRuneAbilityDamage() * 0.65
	spellCoeff := 0.429
	manaCost := 0.08

	damageModFlat := mage.AddDynamicMod(core.SpellModConfig{
		ClassMask:  ClassSpellMask_MageIceLance,
		Kind:       core.SpellMod_DamageDone_Flat,
		FloatValue: 0,
	})

	damageModPct := mage.AddDynamicMod(core.SpellModConfig{
		ClassMask:  ClassSpellMask_MageIceLance,
		Kind:       core.SpellMod_DamageDone_Pct,
		FloatValue: 0,
	})

	mage.IceLance = mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: int32(proto.MageRune_RuneHandsIceLance)},
		ClassSpellMask: ClassSpellMask_MageIceLance,
		SpellSchool:    core.SpellSchoolFrost,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,

		MissileSpeed: 38,
		MetricSplits: 6,

		ManaCost: core.ManaCostOptions{
			BaseCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				if !hasWintersChillTalent {
					return
				}

				if glaciateAura := mage.GlaciateAuras.Get(mage.CurrentTarget); glaciateAura != nil {
					spell.SetMetricsSplit(glaciateAura.GetStacks())
				}
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)

			damageMultiplier := core.TernaryFloat64(mage.isTargetFrozen(target), 3, 0)
			var glaciateAura *core.Aura
			modifier := 0.0
			if hasWintersChillTalent {
				if glaciateAura = mage.GlaciateAuras.Get(target); hasWintersChillTalent && glaciateAura.IsActive() {
					modifier += 0.20 * float64(glaciateAura.GetStacks())
				}
			}
			damageModPct.UpdateFloatValue(damageMultiplier)
			damageModFlat.UpdateFloatValue(modifier)

			damageModPct.Activate()
			damageModFlat.Activate()
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			damageModPct.Deactivate()
			damageModFlat.Deactivate()

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
				if result.Landed() && glaciateAura != nil {
					glaciateAura.Deactivate(sim)
				}
			})
		},
	})

	if !hasWintersChillTalent {
		return
	}

	mage.GlaciateAuras = mage.NewEnemyAuraArray(func(unit *core.Unit, _ int32) *core.Aura {
		return unit.RegisterAura(core.Aura{
			ActionID:  core.ActionID{SpellID: 1218345},
			Label:     "Glaciate",
			Duration:  time.Second * 15,
			MaxStacks: 5,
		})
	})

	core.MakeProcTriggerAura(&mage.Unit, core.ProcTrigger{
		Name:             "Glaciate Trigger",
		ClassSpellMask:   ClassSpellMask_MageAll ^ ClassSpellMask_MageIceLance,
		Callback:         core.CallbackOnSpellHitDealt,
		SpellSchool:      core.SpellSchoolFrost,
		Outcome:          core.OutcomeLanded,
		CanProcFromProcs: true,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			glaciateAura := mage.GlaciateAuras.Get(result.Target)
			glaciateAura.Activate(sim)
			glaciateAura.AddStack(sim)
		},
	})
}
