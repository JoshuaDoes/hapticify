package main

type Filter struct {
	ff       *FFmpeg
	chanIn   []int           //the input channels to use from the source
	chanOut  []int           //the output channels to use for the destination
	settings []FilterSetting //the chain of filters to apply
}

func (ff *FFmpeg) NewFilter() *Filter {
	f := new(Filter)
	f.ff = ff
	f.chanIn = make([]int, 0)
	f.chanOut = make([]int, 0)
	f.settings = make([]FilterSetting, 0)

	ff.filters = append(ff.filters, f)
	return f
}

func (f *Filter) Add(fs ...FilterSetting) {
	if f.ff.precision != "" {
		for i := 0; i < len(fs); i++ {
			fs[i].Set("r", f.ff.precision)
		}
	}
	f.settings = append(f.settings, fs...)
}

func (f *Filter) SetInputChannels(channels ...int) {
	f.chanIn = channels
}
func (f *Filter) SetOutputChannels(channels ...int) {
	f.chanOut = channels
}

func (f *Filter) SetPrecision(precision string) {
	if len(f.settings) > 0 {
		for i := 0; i < len(f.settings); i++ {
			f.settings[i].Set("r", precision)
		}
	}
}
