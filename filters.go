package main

import (
	"fmt"
	"slices"
)

type FilterSetting interface {
	Name() string
	Get(key string) (val string)
	Set(key, val string, vals ...any)
	Settings() []string
}

type filterBase struct {
	vals map[string]string
}

func newFilterBase() *filterBase {
	f := new(filterBase)
	f.vals = make(map[string]string)
	return f
}

func (f *filterBase) Name() string {
	return ""
}

func (f *filterBase) Get(key string) string {
	if val, ok := f.vals[key]; ok {
		return val
	}
	return ""
}
func (f *filterBase) Set(key, val string, vals ...any) {
	if len(vals) > 0 {
		f.vals[key] = fmt.Sprintf(val, vals...)
	} else {
		f.vals[key] = val
	}
}
func (f *filterBase) Settings() []string {
	keys := make([]string, 0)
	for key := range f.vals {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

type FilterHighpass struct {
	*filterBase
}

func (f *FilterHighpass) Name() string {
	return "highpass"
}

func NewFilterHighpass(freq int) *FilterHighpass {
	f := new(FilterHighpass)
	f.filterBase = newFilterBase()
	f.Set("f", "%d", freq)
	return f
}

type FilterLowpass struct {
	*filterBase
}

func (f *FilterLowpass) Name() string {
	return "lowpass"
}

func NewFilterLowpass(freq int) *FilterLowpass {
	f := new(FilterLowpass)
	f.filterBase = newFilterBase()
	f.Set("f", "%d", freq)
	return f
}

type FilterEqualizer struct {
	*filterBase
}

func (f *FilterEqualizer) Name() string {
	return "equalizer"
}

func NewFilterEqualizer(freq, width int, gain float64) *FilterEqualizer {
	f := new(FilterEqualizer)
	f.filterBase = newFilterBase()
	f.Set("f", "%d", freq)
	f.Set("w", "%d", width)
	f.Set("g", "%f", gain)
	return f
}

type FilterVolume struct {
	*filterBase
}

func (f *FilterVolume) Name() string {
	return "volume"
}

func (f *FilterVolume) Set(key, val string, vals ...any) {
	switch key {
	case "r": //Precision does not apply
		return
	}
	f.filterBase.Set(key, val, vals...)
}

func NewFilterVolumePercentage(volume float64) *FilterVolume {
	f := new(FilterVolume)
	f.filterBase = newFilterBase()
	f.Set("volume", "%f", volume)
	return f
}

func NewFilterVolumeGain(gain float64) *FilterVolume {
	f := new(FilterVolume)
	f.filterBase = newFilterBase()
	f.Set("volume", "%fdB", gain)
	return f
}
