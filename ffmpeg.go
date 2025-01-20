package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type FFmpeg struct {
	running   bool
	process   *exec.Cmd
	errors    []error
	onExit    func(ff *FFmpeg)
	onExitRan bool

	input, output           string
	codecIn, codecOut       string
	formatIn, formatOut     string
	channelsIn, channelsOut int
	rateIn, rateOut         int
	bitrateOut              int
	threads                 int
	precision               string
	metadata                map[string]string
	filters                 []*Filter

	stdin, stdout, stderr *Buffer

	buffer  []byte        //Stores the output of the stream until read or flushed
	bufTime time.Duration //length of time to buffer audio for, to keep data in sync with streamers
	bufSize int64         //calculated buffer size of each channel in bytes
}

func NewFFmpeg(codec, format string) *FFmpeg {
	ff := new(FFmpeg)
	ff.errors = make([]error, 0)
	ff.buffer = make([]byte, 0)
	ff.filters = make([]*Filter, 0)
	ff.metadata = make(map[string]string)
	ff.codecOut = codec
	ff.formatOut = format
	ff.stdin = NewBuffer("in")
	ff.stdout = NewBuffer("out")
	ff.stderr = NewBuffer("err")
	return ff
}

func (ff *FFmpeg) Stdin() *Buffer {
	if ff.stdin != nil {
		return ff.stdin.Reference()
	}
	return nil
}
func (ff *FFmpeg) Stdout() *Buffer {
	if ff.stdout != nil {
		return ff.stdout.Reference()
	}
	return nil
}
func (ff *FFmpeg) Stderr() *Buffer {
	if ff.stderr != nil {
		return ff.stderr.Reference()
	}
	return nil
}

// Start begins execution of the ffmpeg process and is non-blocking.
func (ff *FFmpeg) Start() error {
	if ff.IsRunning() {
		return ErrorAlreadyRunning
	}

	process := exec.Command("ffmpeg", ff.Arguments()...)
	//process.Stdin = ff.Stdin()
	process.Stdout = ff.Stdout()
	process.Stderr = ff.Stderr()
	ff.process = process

	ff.spawn()
	return nil
}

func (ff *FFmpeg) spawn() {
	if err := ff.process.Start(); err != nil {
		ff.error(err)
		return
	}
	ff.running = true

	if err := ff.process.Wait(); err != nil {
		ff.error(err)
	}
	ff.running = false

	if ff.onExit != nil {
		ff.onExit(ff)
	}
	ff.onExitRan = true
}

// Close stops the ffmpeg process and cleans up remaining resources.
// Must be called on loop until no error is returned.
func (ff *FFmpeg) Close() error {
	if ff.process != nil {
		if err := ff.process.Process.Kill(); err != nil {
			return err
		}
		ff.process = nil
	}
	ff.running = false
	for {
		//Wait for onExit callback
		if ff.onExitRan {
			break
		}
	}
	return nil
}

// IsRunning returns true if ffmpeg is currently running.
func (ff *FFmpeg) IsRunning() bool {
	if ff.process == nil {
		return false
	}
	return ff.running
}

func (ff *FFmpeg) Wait() {
	for {
		if !ff.IsRunning() {
			break
		}
	}
}

// SetOnExit sets a callback handler for when ffmpeg exits.
func (ff *FFmpeg) SetOnExit(fnc func(ff *FFmpeg)) {
	ff.onExit = fnc
}

func (ff *FFmpeg) SetBufferLength(d time.Duration) {
	ff.bufTime = d
	sampleRate := int64(ff.rateOut)
	channels := int64(ff.channelsOut)
	var bytesPerSample int64

	switch ff.precision {
	case "f64":
		bytesPerSample = 8
	case "f32":
		bytesPerSample = 4
	default:
		bytesPerSample = 4 // Default to 4 bytes per sample
	}

	ff.bufSize = (int64(d) * sampleRate * channels * bytesPerSample) / int64(time.Second)
}

func (ff *FFmpeg) SetBufferSize(n int64) {
	ff.bufSize = n
	sampleRate := int64(ff.rateOut)
	channels := int64(ff.channelsOut)
	var bytesPerSample int64

	switch ff.precision {
	case "f64":
		bytesPerSample = 8
	case "f32":
		bytesPerSample = 4
	default:
		bytesPerSample = 4 // Default to 4 bytes per sample
	}

	ff.bufTime = time.Duration((n * int64(time.Second)) / (sampleRate * channels * bytesPerSample))
}

func (ff *FFmpeg) SetInput(input string) {
	ff.input = input
}
func (ff *FFmpeg) SetOutput(output string) {
	ff.output = output
}
func (ff *FFmpeg) SetInputCodec(codec string) {
	ff.codecIn = codec
}
func (ff *FFmpeg) SetOutputCodec(codec string) {
	ff.codecOut = codec
}
func (ff *FFmpeg) SetInputFormat(format string) {
	ff.formatIn = format
}
func (ff *FFmpeg) SetOutputFormat(format string) {
	ff.formatOut = format
}
func (ff *FFmpeg) SetInputRate(rate int) {
	ff.rateIn = rate
}
func (ff *FFmpeg) SetOutputRate(rate int) {
	ff.rateOut = rate
}
func (ff *FFmpeg) SetOutputBitrate(bitrate int) {
	ff.bitrateOut = bitrate
}
func (ff *FFmpeg) SetThreads(threads int) {
	ff.threads = threads
}
func (ff *FFmpeg) SetPrecision(precision string) {
	ff.precision = precision
	if len(ff.filters) > 0 {
		for i := 0; i < len(ff.filters); i++ {
			ff.filters[i].SetPrecision(precision)
		}
	}
}
func (ff *FFmpeg) SetMetadata(key, value string) {
	ff.metadata[key] = value
}

func (ff *FFmpeg) SetInputChannels(channels int) {
	ff.channelsIn = channels
}
func (ff *FFmpeg) SetOutputChannels(channels int) {
	ff.channelsOut = channels
}

func (ff *FFmpeg) Arguments() []string {
	//Prepare list of args
	args := make([]string, 0)
	args = append(args, "-hide_banner", "-stats")

	//Determine real input and output locations
	input := ff.input
	if input == "" {
		input = "-"
	}
	output := ff.output
	if output == "" {
		output = "pipe:1"
	} else {
		args = append(args, "-y") //Overwrite output without asking
	}

	//Input
	if ff.codecIn != "" {
		args = append(args, "-acodec", ff.codecIn)
	}
	if ff.formatIn != "" {
		args = append(args, "-f", ff.formatIn)
	}
	if ff.channelsIn > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", ff.channelsIn))
	}
	if ff.rateIn > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", ff.rateIn))
	}
	args = append(args, "-i", input)

	//Filters
	args = append(args, "-filter_complex")
	args = append(args, ff.generateFilterComplex("a"))
	args = append(args, "-map", "[a]")

	//Metadata
	for key, val := range ff.metadata {
		arg := fmt.Sprintf("%s=%s", key, val)
		args = append(args, "-metadata", arg)
		args = append(args, "-metadata:s:a:0", arg)
	}

	//Output
	args = append(args, "-acodec", ff.codecOut)
	args = append(args, "-f", ff.formatOut)
	if ff.channelsOut > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", ff.channelsOut))
	}
	if ff.rateOut > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", ff.rateOut))
	}
	if ff.bitrateOut > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%d", ff.bitrateOut))
	}
	if ff.threads > 0 {
		args = append(args, "-threads", fmt.Sprintf("%d", ff.threads))
	}
	args = append(args, output)

	return args
}

func (ff *FFmpeg) generateFilterSetting(fs FilterSetting) string {
	str := fs.Name()
	s := fs.Settings()
	if len(s) > 0 {
		str += "="
		for i := 0; i < len(s); i++ {
			v := fs.Get(s[i])
			str += s[i] + "=" + v + ":"
		}
		str = str[:len(str)-1]
	}
	return str
}

func (ff *FFmpeg) generateFilterComplex(name string) string {
	filters := make([]string, len(ff.filters))

	fc := "[0:a]asplit"
	for i := 0; i < len(ff.filters); i++ {
		filters[i] = fmt.Sprintf("[f%d_0]", i)
		fc += filters[i]
	}
	fc += ";"

	for i := 0; i < len(ff.filters); i++ {
		f := ff.filters[i]
		for j := 0; j < len(f.settings); j++ {
			next := fmt.Sprintf("[f%d_%d]", i, j+1)
			fs := ff.generateFilterSetting(f.settings[j])
			fc += fmt.Sprintf("%s%s%s;", filters[i], fs, next)
			filters[i] = next
		}
	}

	inputs := ""
	pan := make([]string, 0)
	for i := 0; i < len(ff.filters); i++ {
		inputs += filters[i]

		f := ff.filters[i]
		for j := 0; j < len(f.chanOut); j++ {
			pan = append(pan, fmt.Sprintf("c%d=c%d", len(pan), f.chanOut[j]))
		}
	}
	fc += fmt.Sprintf("%samerge=inputs=%d,pan=%dc|%s[%s]", inputs, len(ff.filters), ff.channelsOut, strings.Join(pan, "|"), name)

	return fc
}

func (ff *FFmpeg) String() string {
	return fmt.Sprintf("ffmpeg\n%s", strings.Join(ff.Arguments(), "\n"))
}

func (ff *FFmpeg) error(err error) {
	if err != nil {
		ff.errors = append(ff.errors, err)
	}
}

func (ff *FFmpeg) Error() error {
	errs := ""
	for i := 0; i < len(ff.errors); i++ {
		errs += fmt.Sprintf("%v\n", ff.errors[i])
	}
	if errs == "" {
		return nil
	}
	errs = errs[:len(errs)-1]
	return fmt.Errorf("%s", errs)
}
