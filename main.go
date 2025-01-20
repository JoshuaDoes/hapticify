package main

import (
	"github.com/spf13/pflag"

	"fmt"
	"runtime"
	"time"
)

var (
	ErrorAlreadyRunning error = fmt.Errorf("ffmpeg: already running")
)

var (
	in, out   string
	high, low int
	vol, hap  float64
)

func main() {
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&in, "in", "i", "", "input audio")
	pflag.StringVarP(&out, "out", "o", "", "output audio")
	pflag.IntVarP(&high, "high", "h", 70, "highpass freq")
	pflag.IntVarP(&low, "low", "l", 150, "lowpass freq")
	pflag.Float64VarP(&vol, "vol", "v", 1, "audio dB gain")
	pflag.Float64VarP(&vol, "hap", "s", 1, "haptics dB gain")
	pflag.Parse()

	if in == "" {
		panic("need input")
	}
	if out == "" {
		out = in + ".hapticify.ogg"
	}

	ff := NewFFmpeg("libvorbis", "ogg")       //Android haptics must be OGG Vorbis
	ff.SetInput(in)                           //Input file
	ff.SetOutput(out)                         //Output file
	ff.SetMetadata("HAPTICIFY", "JoshuaDoes") //:D
	ff.SetMetadata("ANDROID_HAPTIC", "2")     //Tell Android we have stereo haptics
	ff.SetInputChannels(2)                    //Mix input to stereo
	ff.SetOutputChannels(4)                   //Mix output to stereo audio + stereo haptics
	ff.SetOutputBitrate(960000)               //960Kbps = 480Kbps audio + 480Kbps haptics, 240Kbps per channel
	ff.SetOutputRate(48000)                   //48KHz audio + haptics
	ff.SetPrecision("f64")                    //Use 64-bit floating point precision for filters and mixing
	ff.SetThreads(runtime.NumCPU())           //Use all CPU cores
	ff.SetBufferLength(time.Millisecond * 20) //20ms audio buffer for low latency

	hp := ff.NewFilter()
	hp.SetInputChannels(0, 1)
	hp.SetOutputChannels(0, 1)
	hp.Add(NewFilterHighpass(high))
	hp.Add(NewFilterVolumeGain(vol))

	lp := ff.NewFilter()
	lp.SetInputChannels(0, 1)
	lp.SetOutputChannels(2, 3)
	lp.Add(NewFilterLowpass(low))
	lp.Add(NewFilterVolumeGain(hap))

	//Set a function to run as soon as ffmpeg exits
	ff.SetOnExit(func(ff *FFmpeg) {
		if err := ff.Error(); err != nil {
			fmt.Printf("ERRORS ENCOUNTERED: %v\n\n", err)
		}
		stdout := ff.Stdout().Bytes()
		fmt.Printf("STDOUT: %d bytes\n%s\n\n", len(stdout), string(stdout))
		stderr := ff.Stderr().Bytes()
		fmt.Printf("STDERR: %d bytes\n%s\n\n", len(stderr), string(stderr))
	})

	//Start the conversion process
	if err := ff.Start(); err != nil {
		panic(err)
	}

	//Wait until it finishes
	ff.Wait()
}
