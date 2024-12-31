package druid

import (
	"time"

	"github.com/wowsims/sod/sim/core"
)

func (druid *Druid) registerFaerieFireSpell() {
	spellCode := SpellCode_DruidFaerieFire
	actionID := core.ActionID{SpellID: map[int32]int32{
		25: 770,
		40: 778,
		50: 9749,
		60: 9907,
	}[druid.Level]}
	manaCostOptions := core.ManaCostOptions{
		FlatCost: map[int32]float64{
			25: 55,
			40: 75,
			50: 95,
			60: 115,
		}[druid.Level],
	}
	gcd := core.GCDDefault
	ignoreHaste := false
	cd := core.Cooldown{}
	flatThreatBonus := 2. * map[int32]float64{
		25: 18,
		40: 30,
		50: 42,
		60: 54,
	}[druid.Level]
	flags := core.SpellFlagNone
	formMask := Humanoid | Moonkin

	druid.FaerieFireAuras = druid.NewEnemyAuraArray(func(target *core.Unit, level int32) *core.Aura {
		return core.FaerieFireAura(target, level)
	})

	if druid.InForm(Cat|Bear) && druid.Talents.FaerieFireFeral {
		spellCode = SpellCode_DruidFaerieFireFeral
		actionID = core.ActionID{SpellID: map[int32]int32{
			40: 17390,
			50: 17391,
			60: 17392,
		}[druid.Level]}
		manaCostOptions = core.ManaCostOptions{}
		gcd = time.Second
		ignoreHaste = true
		formMask = Cat | Bear
		cd = core.Cooldown{
			Timer:    druid.NewTimer(),
			Duration: time.Second * 6,
		}
		druid.FaerieFireAuras = druid.NewEnemyAuraArray(func(target *core.Unit, level int32) *core.Aura {
			return core.FaerieFireFeralAura(target, level)
		})
	}
	flags |= core.SpellFlagAPL

	druid.FaerieFire = druid.RegisterSpell(formMask, core.SpellConfig{
		SpellCode:   spellCode,
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolNature,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       flags,

		ManaCost: manaCostOptions,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: gcd,
			},
			IgnoreHaste: ignoreHaste,
			CD:          cd,
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  flatThreatBonus,
		DamageMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				druid.FaerieFireAuras.Get(target).Activate(sim)
			}

			if druid.InForm(Humanoid | Moonkin) {
				druid.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime, true, false)
			}
		},

		RelatedAuras: []core.AuraArray{druid.FaerieFireAuras},
	})
}

func (druid *Druid) ShouldFaerieFire(sim *core.Simulation, target *core.Unit) bool {
	if druid.FaerieFire == nil {
		return false
	}

	if !druid.FaerieFire.IsReady(sim) {
		return false
	}

	debuff := druid.FaerieFireAuras.Get(target)
	return !debuff.IsActive() || debuff.RemainingDuration(sim) < time.Second*4
}
