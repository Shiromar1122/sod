package core

import (
	"fmt"
	"time"

	"github.com/wowsims/sod/sim/core/proto"
)

const MaxRage = 100.0
const ThreatPerRageGained = 5

// OnRageChange is called any time rage is increased.
type OnRageChange func(aura *Aura, sim *Simulation, metrics *ResourceMetrics)

type rageBar struct {
	unit *Unit

	damageDealtMultiplier float64 // Multiplier for rage generation from damage dealt
	damageTakenMultiplier float64 // Multiplier for rage generation from damage taken

	flatDamageDealtBonusRage float64
	flatDamageTakenBonusRage float64

	startingRage float64
	currentRage  float64

	RageRefundMetrics *ResourceMetrics
}

type RageBarOptions struct {
	StartingRage          float64
	DamageDealtMultiplier float64
	DamageTakenMultiplier float64
}

func GetRageConversion(attacker_level int32) float64 {
	if attacker_level == 25 {
		return 82.25 // Tested
	} else if attacker_level == 40 {
		return 140.5 // Tested
	} else if attacker_level < 45 {
		// Poor fit, but better then current formula below 45
		return 0.0215*float64(attacker_level^2) + 2.66*float64(attacker_level) + 0.89
	} else {
		// Rage conversion is adjusted according to target stats (https://web.archive.org/web/20201118213002/https://blue.mmo-champion.com/topic/18325-the-new-rage-formula-by-kalgan/)\
		// So this is probably only the base value formula and will be slightly wrong for most target
		return 0.0091107836*float64(attacker_level^2) + 3.225598133*float64(attacker_level) + 4.2652911
	}
}

func (unit *Unit) EnableRageBar(options RageBarOptions) {
	rageFromDamageTakenMetrics := unit.NewRageMetrics(ActionID{OtherID: proto.OtherAction_OtherActionDamageTaken})
	rageConversion := GetRageConversion(unit.Level)

	unit.SetCurrentPowerBar(RageBar)
	unit.RegisterAura(Aura{
		Label:    "RageBar",
		Duration: NeverExpires,
		OnInit: func(aura *Aura, sim *Simulation) {
			// Initialize resource metrics for rage gain for auto attacks here to make sure tag is correct.
			// Extra attacks change the tag from 1 to 3 for mh hits.
			mhSpell := unit.AutoAttacks.MHAuto()
			if mhSpell != nil {
				mhSpell.ResourceMetrics = unit.NewRageMetrics(mhSpell.ActionID)
			}
			ohSpell := unit.AutoAttacks.OHAuto()
			if ohSpell != nil {
				ohSpell.ResourceMetrics = unit.NewRageMetrics(ohSpell.ActionID)
			}
		},
		OnReset: func(aura *Aura, sim *Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if unit.GetCurrentPowerBar() != RageBar {
				return
			}
			if result.Outcome.Matches(OutcomeMiss) {
				return
			}
			if spell.ProcMask != ProcMaskMeleeMHAuto && spell.ProcMask != ProcMaskMeleeOHAuto {
				return
			}

			damage := result.Damage
			if result.Outcome.Matches(OutcomeDodge | OutcomeParry) {
				// Rage is still generated for dodges/parries, based on the damage it WOULD have done.
				damage = result.PreOutcomeDamage
			}

			generatedRage := damage * 7.5 / rageConversion
			generatedRage *= unit.rageBar.damageDealtMultiplier
			generatedRage += unit.rageBar.flatDamageDealtBonusRage

			var metrics *ResourceMetrics
			if spell.Cost != nil {
				metrics = spell.Cost.SpellCostFunctions.(*RageCost).ResourceMetrics
			} else {
				// Seems like only auto attacks are using this. See OnInit handler of this aura.
				if spell.ResourceMetrics == nil {
					panic(fmt.Sprintf("Spell ResourceMetrics are nil for spell %v", spell.ActionID))
				}
				metrics = spell.ResourceMetrics
			}
			unit.AddRage(sim, generatedRage, metrics)
		},
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if unit.GetCurrentPowerBar() != RageBar {
				return
			}
			rageConversionDamageTaken := GetRageConversion(spell.Unit.Level)
			generatedRage := result.Damage * 2.5 / rageConversionDamageTaken
			generatedRage *= unit.rageBar.damageTakenMultiplier
			generatedRage += unit.rageBar.flatDamageTakenBonusRage
			unit.AddRage(sim, generatedRage, rageFromDamageTakenMetrics)
		},
	})

	// Not a real spell, just holds metrics from rage gain threat.
	unit.RegisterSpell(SpellConfig{
		ActionID: ActionID{OtherID: proto.OtherAction_OtherActionRageGain},
	})

	unit.rageBar = rageBar{
		unit:                  unit,
		damageDealtMultiplier: options.DamageDealtMultiplier,
		damageTakenMultiplier: options.DamageTakenMultiplier,
		startingRage:          max(0, min(options.StartingRage, MaxRage)),
		RageRefundMetrics:     unit.NewRageMetrics(ActionID{OtherID: proto.OtherAction_OtherActionRefund}),
	}
}

func (unit *Unit) HasRageBar() bool {
	return unit.rageBar.unit != nil
}

func (unit *Unit) AddDamageDealtRageMultiplier(multi float64) {
	unit.rageBar.damageDealtMultiplier *= multi
}

func (unit *Unit) AddDamageTakenRageMultiplier(multi float64) {
	unit.rageBar.damageTakenMultiplier *= multi
}

func (unit *Unit) AddDamageDealtRageBonus(bonus float64) {
	unit.rageBar.flatDamageDealtBonusRage += bonus
}

func (unit *Unit) AddDamageTakenRageBonus(bonus float64) {
	unit.rageBar.flatDamageTakenBonusRage += bonus
}

func (rb *rageBar) CurrentRage() float64 {
	return rb.currentRage
}

func (rb *rageBar) AddRage(sim *Simulation, amount float64, metrics *ResourceMetrics) {
	if amount < 0 {
		panic("Trying to add negative rage!")
	}

	newRage := min(rb.currentRage+amount, MaxRage)
	metrics.AddEvent(amount, newRage-rb.currentRage)

	if sim.Log != nil {
		rb.unit.Log(sim, "Gained %0.3f rage from %s (%0.3f --> %0.3f).", amount, metrics.ActionID, rb.currentRage, newRage)
	}

	rb.currentRage = newRage
	if !sim.Options.Interactive {
		rb.unit.ReactToEvent(sim)
	}
	StartDelayedAction(sim, DelayedActionOptions{
		DoAt: sim.CurrentTime + time.Millisecond*1,
		OnAction: func(sim *Simulation) {
			rb.unit.OnRageChange(sim, metrics)
		},
	})

}

func (rb *rageBar) SpendRage(sim *Simulation, amount float64, metrics *ResourceMetrics) {
	if amount < 0 {
		panic("Trying to spend negative rage!")
	}

	newRage := rb.currentRage - amount
	metrics.AddEvent(-amount, -amount)

	if sim.Log != nil {
		rb.unit.Log(sim, "Spent %0.3f rage from %s (%0.3f --> %0.3f).", amount, metrics.ActionID, rb.currentRage, newRage)
	}

	rb.currentRage = newRage

	rb.unit.OnRageChange(sim, metrics)
}

func (rb *rageBar) reset(_ *Simulation) {
	if rb.unit == nil {
		return
	}

	rb.currentRage = rb.startingRage
}

func (rb *rageBar) doneIteration() {
	if rb.unit == nil {
		return
	}

	rageGainSpell := rb.unit.GetSpell(ActionID{OtherID: proto.OtherAction_OtherActionRageGain})

	for _, resourceMetrics := range rb.unit.Metrics.resources {
		if resourceMetrics.Type != proto.ResourceType_ResourceTypeRage {
			continue
		}
		if resourceMetrics.ActionID.SameActionIgnoreTag(ActionID{OtherID: proto.OtherAction_OtherActionDamageTaken}) {
			continue
		}
		if resourceMetrics.ActionID.SameActionIgnoreTag(ActionID{OtherID: proto.OtherAction_OtherActionRefund}) {
			continue
		}
		if resourceMetrics.ActualGainForCurrentIteration() <= 0 {
			continue
		}

		// Need to exclude rage gained from white hits. Rather than have a manual list of all IDs that would
		// apply here (autos, WF attack, sword spec procs, etc), just check if the effect caused any damage.
		sourceSpell := rb.unit.GetSpell(resourceMetrics.ActionID)
		if sourceSpell != nil && sourceSpell.SpellMetrics[0].TotalDamage > 0 {
			continue
		}

		rageGainSpell.SpellMetrics[0].Casts += resourceMetrics.EventsForCurrentIteration()
		rageGainSpell.ApplyAOEThreatIgnoreMultipliers(resourceMetrics.ActualGainForCurrentIteration() * ThreatPerRageGained)
	}
}

type RageCostOptions struct {
	Cost float64

	Refund        float64
	RefundMetrics *ResourceMetrics // Optional, will default to unit.RageRefundMetrics if not supplied.
}
type RageCost struct {
	Refund          float64
	RefundMetrics   *ResourceMetrics
	ResourceMetrics *ResourceMetrics
}

func newRageCost(spell *Spell, options RageCostOptions) *SpellCost {
	if options.Refund > 0 && options.RefundMetrics == nil {
		options.RefundMetrics = spell.Unit.RageRefundMetrics
	}

	return &SpellCost{
		spell:      spell,
		BaseCost:   options.Cost,
		Multiplier: 100,
		SpellCostFunctions: &RageCost{
			Refund:          options.Refund * options.Cost,
			RefundMetrics:   options.RefundMetrics,
			ResourceMetrics: spell.Unit.NewRageMetrics(spell.ActionID),
		},
	}
}

func (rc *RageCost) CostType() CostType {
	return CostTypeRage
}

func (rc *RageCost) MeetsRequirement(_ *Simulation, spell *Spell) bool {
	spell.CurCast.Cost = spell.Cost.GetCurrentCost()
	return spell.Unit.CurrentRage() >= spell.CurCast.Cost
}
func (rc *RageCost) CostFailureReason(sim *Simulation, spell *Spell) string {
	return fmt.Sprintf("not enough rage (Current Rage = %0.03f, Rage Cost = %0.03f)", spell.Unit.CurrentRage(), spell.CurCast.Cost)
}
func (rc *RageCost) SpendCost(sim *Simulation, spell *Spell) {
	if spell.CurCast.Cost > 0 {
		spell.Unit.SpendRage(sim, spell.CurCast.Cost, rc.ResourceMetrics)
	}
}
func (rc *RageCost) IssueRefund(sim *Simulation, spell *Spell) {
	if rc.Refund > 0 {
		spell.Unit.AddRage(sim, rc.Refund, rc.RefundMetrics)
	}
}
