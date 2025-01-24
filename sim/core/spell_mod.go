package core

import (
	"strconv"
	"time"
)

/*
SpellMod implementation.
*/

type SpellModConfig struct {
	ClassMask    int64
	Kind         SpellModType
	School       SpellSchool
	ProcMask     ProcMask
	IntValue     int64
	TimeValue    time.Duration
	FloatValue   float64
	KeyValue     string
	ApplyCustom  SpellModApply
	RemoveCustom SpellModRemove
}

type SpellMod struct {
	ClassMask      int64
	Kind           SpellModType
	School         SpellSchool
	ProcMask       ProcMask
	floatValue     float64
	intValue       int64
	timeValue      time.Duration
	keyValue       string
	Apply          SpellModApply
	Remove         SpellModRemove
	IsActive       bool
	AffectedSpells []*Spell
}

type SpellModApply func(mod *SpellMod, spell *Spell)
type SpellModRemove func(mod *SpellMod, spell *Spell)
type SpellModFunctions struct {
	Apply  SpellModApply
	Remove SpellModRemove
}

func buildMod(unit *Unit, config SpellModConfig) *SpellMod {
	functions := spellModMap[config.Kind]
	if functions == nil {
		panic("SpellMod " + strconv.Itoa(int(config.Kind)) + " not implmented")
	}

	var applyFn SpellModApply
	var removeFn SpellModRemove

	if config.Kind == SpellMod_Custom {
		if (config.ApplyCustom == nil) || (config.RemoveCustom == nil) {
			panic("ApplyCustom and RemoveCustom are mandatory fields for SpellMod_Custom")
		}

		applyFn = config.ApplyCustom
		removeFn = config.RemoveCustom
	} else {
		applyFn = functions.Apply
		removeFn = functions.Remove
	}

	mod := &SpellMod{
		ClassMask:  config.ClassMask,
		Kind:       config.Kind,
		School:     config.School,
		ProcMask:   config.ProcMask,
		floatValue: config.FloatValue,
		intValue:   config.IntValue,
		timeValue:  config.TimeValue,
		keyValue:   config.KeyValue,
		Apply:      applyFn,
		Remove:     removeFn,
		IsActive:   false,
	}

	unit.OnSpellRegistered(func(spell *Spell) {
		if shouldApply(spell, mod) {
			mod.AffectedSpells = append(mod.AffectedSpells, spell)

			if mod.IsActive {
				mod.Apply(mod, spell)
			}
		}
	})

	return mod
}

func (unit *Unit) AddStaticMod(config SpellModConfig) {
	mod := buildMod(unit, config)
	mod.Activate()
}

func (unit *Unit) AddDynamicMod(config SpellModConfig) *SpellMod {
	return buildMod(unit, config)
}

func shouldApply(spell *Spell, mod *SpellMod) bool {
	if spell.Flags.Matches(SpellFlagNoSpellMods) {
		return false
	}

	if mod.ClassMask > 0 && !spell.Matches(mod.ClassMask) {
		return false
	}

	if mod.School > 0 && !mod.School.Matches(spell.SpellSchool) {
		return false
	}

	if mod.ProcMask > 0 && !mod.ProcMask.Matches(spell.ProcMask) {
		return false
	}

	return true
}

func (mod *SpellMod) UpdateIntValue(value int64) {
	if mod.IsActive {
		mod.Deactivate()
		mod.intValue = value
		mod.Activate()
	} else {
		mod.intValue = value
	}
}

func (mod *SpellMod) UpdateTimeValue(value time.Duration) {
	if mod.IsActive {
		mod.Deactivate()
		mod.timeValue = value
		mod.Activate()
	} else {
		mod.timeValue = value
	}
}

func (mod *SpellMod) UpdateFloatValue(value float64) {
	if mod.IsActive {
		mod.Deactivate()
		mod.floatValue = value
		mod.Activate()
	} else {
		mod.floatValue = value
	}
}

func (mod *SpellMod) GetIntValue() int64 {
	return mod.intValue
}

func (mod *SpellMod) GetFloatValue() float64 {
	return mod.floatValue
}

func (mod *SpellMod) GetTimeValue() time.Duration {
	return mod.timeValue
}

func (mod *SpellMod) Activate() {
	if mod.IsActive {
		return
	}

	for _, spell := range mod.AffectedSpells {
		mod.Apply(mod, spell)
	}

	mod.IsActive = true
}

func (mod *SpellMod) Deactivate() {
	if !mod.IsActive {
		return
	}

	for _, spell := range mod.AffectedSpells {
		mod.Remove(mod, spell)
	}

	mod.IsActive = false
}

// Mod implmentations
type SpellModType uint32

const (
	// Will multiply the spell.DamageDoneMultiplier. +5% = 0.05
	// Uses FloatValue
	SpellMod_DamageDone_Pct SpellModType = 1 << iota

	// Will add the value spell.DamageMultiplierAdditive
	// Uses FloatValue
	SpellMod_DamageDone_Flat

	// Will add the value spell.BaseDamageMultiplierAdditive
	// Uses FloatValue
	SpellMod_BaseDamageDone_Flat

	// Will add the value spell.PeriodicDamageMultiplierAdditive
	// Uses FloatValue
	SpellMod_PeriodicDamageDone_Flat

	// Will add the value spell.ImpactDamageMultiplierAdditive
	// Uses FloatValue
	SpellMod_ImpactDamageDone_Flat

	// Will add the value spell.CritDamageBonus
	// Uses FloatValue
	SpellMod_CritDamageBonus_Flat

	// Will reduce spell.DefaultCast.Cost by % amount. -5% = -0.05
	// Uses IntValue
	SpellMod_PowerCost_Pct

	// Increases or decreases spell.DefaultCast.Cost by flat amount
	// Uses IntValue
	SpellMod_PowerCost_Flat

	// Will add time.Duration to spell.CD.FlatModifier
	// Uses TimeValue
	SpellMod_Cooldown_Flat

	// Increases or decreases spell.CD.Multiplier by flat amount
	// Uses FloatValue
	SpellMod_Cooldown_Multi_Flat

	// Increases or decreases spell.CD.Multiplier by % amount. +50% = 0.5
	// Uses FloatValue
	SpellMod_Cooldown_Multi_Pct

	// Add/subtract BonusCritRating. +1% = 1.0
	// Uses: FloatValue
	SpellMod_BonusCrit_Flat

	// Add/subtract BonusHitRating. +1% = 1.0
	// Uses: FloatValue
	SpellMod_BonusHit_Flat

	// Will add / substract % amount from the cast time multiplier.
	// Ueses: FloatValue
	SpellMod_CastTime_Pct

	// Will add / substract time from the cast time.
	// Ueses: TimeValue
	SpellMod_CastTime_Flat

	// Add/subtract to the dots max ticks
	// Uses: IntValue
	SpellMod_DotNumberOfTicks_Flat

	// Add/substrct to the base tick frequency
	// Uses: TimeValue
	SpellMod_DotTickLength_Flat

	// Add/subtract to the casts gcd
	// Uses: TimeValue
	SpellMod_GlobalCooldown_Flat

	// Add/subtract bonus coefficient
	// Uses: FloatValue
	SpellMod_BonusCoeffecient_Flat

	// Enables casting while moving
	SpellMod_AllowCastWhileMoving

	// Add/subtract bonus spell power
	// Uses: FloatValue
	SpellMod_BonusDamage_Flat

	// Add/subtract bonus expertise rating
	// Uses: FloatValue
	SpellMod_BonusExpertise_Rating

	// Add/subtract duration for associated debuff
	// Uses: KeyValue, TimeValue
	SpellMod_DebuffDuration_Flat

	// Add/subtract duration for associated self-buff
	// Uses: TimeValue
	SpellMod_BuffDuration_Flat

	// User-defined implementation
	// Uses: ApplyCustom | RemoveCustom
	SpellMod_Custom
)

var spellModMap = map[SpellModType]*SpellModFunctions{
	SpellMod_DamageDone_Pct: {
		Apply:  applyDamageDonePercent,
		Remove: removeDamageDonePercent,
	},

	SpellMod_DamageDone_Flat: {
		Apply:  applyDamageDoneAdd,
		Remove: removeDamageDoneAdd,
	},

	SpellMod_BaseDamageDone_Flat: {
		Apply:  applyBaseDamageDoneAdd,
		Remove: removeBaseDamageDoneAdd,
	},

	SpellMod_PeriodicDamageDone_Flat: {
		Apply:  applyPeriodicDamageDoneAdd,
		Remove: removePeriodicDamageDoneAdd,
	},

	SpellMod_ImpactDamageDone_Flat: {
		Apply:  applyImpactDamageDoneAdd,
		Remove: removeImpactDamageDoneAdd,
	},

	SpellMod_CritDamageBonus_Flat: {
		Apply:  applyCritDamageBonusAdd,
		Remove: removeCritDamageBonusAdd,
	},

	SpellMod_PowerCost_Pct: {
		Apply:  applyPowerCostPercent,
		Remove: removePowerCostPercent,
	},

	SpellMod_PowerCost_Flat: {
		Apply:  applyPowerCostFlat,
		Remove: removePowerCostFlat,
	},

	SpellMod_Cooldown_Flat: {
		Apply:  applyCooldownFlat,
		Remove: removeCooldownFlat,
	},

	SpellMod_Cooldown_Multi_Flat: {
		Apply:  applyCooldownMultiplierFlat,
		Remove: removeCooldownMultiplierFlat,
	},

	SpellMod_Cooldown_Multi_Pct: {
		Apply:  applyCooldownMultiplierPct,
		Remove: removeCooldownMultiplierPct,
	},

	SpellMod_CastTime_Pct: {
		Apply:  applyCastTimePercent,
		Remove: removeCastTimePercent,
	},

	SpellMod_CastTime_Flat: {
		Apply:  applyCastTimeFlat,
		Remove: removeCastTimeFlat,
	},

	SpellMod_BonusCrit_Flat: {
		Apply:  applyBonusCritFlat,
		Remove: removeBonusCritFlat,
	},

	SpellMod_BonusHit_Flat: {
		Apply:  applyBonusHitFlat,
		Remove: removeBonusHitFlat,
	},

	SpellMod_DotNumberOfTicks_Flat: {
		Apply:  applyDotNumberOfTicks,
		Remove: removeDotNumberOfTicks,
	},

	SpellMod_DotTickLength_Flat: {
		Apply:  applyDotTickLengthFlat,
		Remove: removeDotTickLengthFlat,
	},

	SpellMod_GlobalCooldown_Flat: {
		Apply:  applyGlobalCooldownFlat,
		Remove: removeGlobalCooldownFlat,
	},

	SpellMod_BonusCoeffecient_Flat: {
		Apply:  applyBonusCoefficientFlat,
		Remove: removeBonusCoefficientFlat,
	},

	SpellMod_BonusDamage_Flat: {
		Apply:  applyBonusDamageFlat,
		Remove: removeBonusDamageFlat,
	},

	SpellMod_Custom: {
		// Doesn't have dedicated Apply/Remove functions as ApplyCustom/RemoveCustom is handled in buildMod()
	},
}

func applyDamageDonePercent(mod *SpellMod, spell *Spell) {
	spell.DamageMultiplier *= 1 + mod.floatValue
}

func removeDamageDonePercent(mod *SpellMod, spell *Spell) {
	spell.DamageMultiplier /= 1 + mod.floatValue
}

func applyDamageDoneAdd(mod *SpellMod, spell *Spell) {
	spell.DamageMultiplierAdditive += mod.floatValue
}

func removeDamageDoneAdd(mod *SpellMod, spell *Spell) {
	spell.DamageMultiplierAdditive -= mod.floatValue
}

func applyBaseDamageDoneAdd(mod *SpellMod, spell *Spell) {
	spell.BaseDamageMultiplierAdditive += mod.floatValue
}

func removeBaseDamageDoneAdd(mod *SpellMod, spell *Spell) {
	spell.BaseDamageMultiplierAdditive -= mod.floatValue
}

func applyPeriodicDamageDoneAdd(mod *SpellMod, spell *Spell) {
	if len(spell.Dots()) > 0 {
		spell.PeriodicDamageMultiplierAdditive += mod.floatValue
	}
}

func removePeriodicDamageDoneAdd(mod *SpellMod, spell *Spell) {
	if len(spell.Dots()) > 0 {
		spell.PeriodicDamageMultiplierAdditive -= mod.floatValue
	}
}

func applyImpactDamageDoneAdd(mod *SpellMod, spell *Spell) {
	spell.ImpactDamageMultiplierAdditive += mod.floatValue
}

func removeImpactDamageDoneAdd(mod *SpellMod, spell *Spell) {
	spell.ImpactDamageMultiplierAdditive -= mod.floatValue
}

func applyCritDamageBonusAdd(mod *SpellMod, spell *Spell) {
	spell.CritDamageBonus += mod.floatValue
}

func removeCritDamageBonusAdd(mod *SpellMod, spell *Spell) {
	spell.CritDamageBonus -= mod.floatValue
}

func applyPowerCostPercent(mod *SpellMod, spell *Spell) {
	if spell.Cost != nil {
		spell.Cost.Multiplier += int32(mod.intValue)
	}
}

func removePowerCostPercent(mod *SpellMod, spell *Spell) {
	if spell.Cost != nil {
		spell.Cost.Multiplier -= int32(mod.intValue)
	}
}

func applyPowerCostFlat(mod *SpellMod, spell *Spell) {
	if spell.Cost != nil {
		spell.Cost.FlatModifier += int32(mod.intValue)
	}
}

func removePowerCostFlat(mod *SpellMod, spell *Spell) {
	if spell.Cost != nil {
		spell.Cost.FlatModifier -= int32(mod.intValue)
	}
}

func applyCooldownFlat(mod *SpellMod, spell *Spell) {
	spell.CD.FlatModifier += mod.timeValue
}

func removeCooldownFlat(mod *SpellMod, spell *Spell) {
	spell.CD.FlatModifier -= mod.timeValue
}

func applyCooldownMultiplierFlat(mod *SpellMod, spell *Spell) {
	spell.CD.Multiplier += mod.floatValue
}

func removeCooldownMultiplierFlat(mod *SpellMod, spell *Spell) {
	spell.CD.Multiplier -= mod.floatValue
}

func applyCooldownMultiplierPct(mod *SpellMod, spell *Spell) {
	spell.CD.Multiplier *= 1 + mod.floatValue
}

func removeCooldownMultiplierPct(mod *SpellMod, spell *Spell) {
	spell.CD.Multiplier /= 1 + mod.floatValue
}

func applyCastTimePercent(mod *SpellMod, spell *Spell) {
	spell.CastTimeMultiplier += mod.floatValue
}

func removeCastTimePercent(mod *SpellMod, spell *Spell) {
	spell.CastTimeMultiplier -= mod.floatValue
}

func applyCastTimeFlat(mod *SpellMod, spell *Spell) {
	spell.DefaultCast.CastTime += mod.timeValue
}

func removeCastTimeFlat(mod *SpellMod, spell *Spell) {
	spell.DefaultCast.CastTime -= mod.timeValue
}

func applyBonusCritFlat(mod *SpellMod, spell *Spell) {
	spell.BonusCritRating += mod.floatValue
}

func removeBonusCritFlat(mod *SpellMod, spell *Spell) {
	spell.BonusCritRating -= mod.floatValue
}

func applyBonusHitFlat(mod *SpellMod, spell *Spell) {
	spell.BonusHitRating += mod.floatValue
}

func removeBonusHitFlat(mod *SpellMod, spell *Spell) {
	spell.BonusHitRating -= mod.floatValue
}

func applyDotNumberOfTicks(mod *SpellMod, spell *Spell) {
	if spell.dots != nil {
		for _, dot := range spell.dots {
			if dot != nil {
				dot.NumberOfTicks += int32(mod.intValue)
			}
		}
	}
	if spell.aoeDot != nil {
		spell.aoeDot.NumberOfTicks += int32(mod.intValue)
	}
}

func removeDotNumberOfTicks(mod *SpellMod, spell *Spell) {
	if spell.dots != nil {
		for _, dot := range spell.dots {
			if dot != nil {
				dot.NumberOfTicks -= int32(mod.intValue)
			}
		}
	}
	if spell.aoeDot != nil {
		spell.aoeDot.NumberOfTicks -= int32(mod.intValue)
	}
}

func applyDotTickLengthFlat(mod *SpellMod, spell *Spell) {
	if spell.dots != nil {
		for _, dot := range spell.dots {
			if dot != nil {
				dot.TickLength += mod.timeValue
			}
		}
	}
	if spell.aoeDot != nil {
		spell.aoeDot.TickLength += mod.timeValue
	}
}

func removeDotTickLengthFlat(mod *SpellMod, spell *Spell) {
	if spell.dots != nil {
		for _, dot := range spell.dots {
			if dot != nil {
				dot.TickLength -= mod.timeValue
			}
		}
	}
	if spell.aoeDot != nil {
		spell.aoeDot.TickLength -= mod.timeValue
	}
}

func applyGlobalCooldownFlat(mod *SpellMod, spell *Spell) {
	spell.DefaultCast.GCD += mod.timeValue
}

func removeGlobalCooldownFlat(mod *SpellMod, spell *Spell) {
	spell.DefaultCast.GCD -= mod.timeValue
}

func applyBonusCoefficientFlat(mod *SpellMod, spell *Spell) {
	spell.BonusCoefficient += mod.floatValue
}

func removeBonusCoefficientFlat(mod *SpellMod, spell *Spell) {
	spell.BonusCoefficient -= mod.floatValue
}
func applyBonusDamageFlat(mod *SpellMod, spell *Spell) {
	spell.BonusDamage += mod.floatValue
}

func removeBonusDamageFlat(mod *SpellMod, spell *Spell) {
	spell.BonusDamage -= mod.floatValue
}
