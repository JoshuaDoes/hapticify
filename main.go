package main

import (
	"github.com/JoshuaDoes/crunchio"
	"github.com/JoshuaDoes/ffmpeg"
	"github.com/spf13/pflag"

	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	in        []string
	high, low int
	vol, hap  float64

	ext = ".hapticify.ogg"
	filterhp  = ffmpeg.NewFilterHighpass(high)
	filtervol = ffmpeg.NewFilterVolumeGain(vol)
	filterlp  = ffmpeg.NewFilterLowpass(low)
	filterhap = ffmpeg.NewFilterVolumeGain(hap)
)

func main() {
	pflag.CommandLine.SortFlags = false
	pflag.StringSliceVarP(&in, "in", "i", []string{"."}, "input audio")
	pflag.StringVarP(&ext, "ext", "e", ext, "output extension")
	pflag.IntVarP(&high, "high", "h", 70, "highpass freq")
	pflag.IntVarP(&low, "low", "l", 150, "lowpass freq")
	pflag.Float64VarP(&vol, "vol", "v", 0, "audio dB gain")
	pflag.Float64VarP(&vol, "hap", "s", 0, "haptics dB gain")
	pflag.Parse()

	if len(in) == 0 {
		panic("need input")
	}

	//Process folders into a file list
	newIn := make([]string, 0)
	for i := 0; i < len(in); i++ {
		newIn = append(newIn, files(in[i])...)
	}

	//Skip over old conversions
	in = make([]string, 0)
	for i := 0; i < len(newIn); i++ {
		if strings.HasSuffix(newIn[i], ext) {
			continue
		}
		in = append(in, newIn[i])
	}

	//Hapticify inputs!
	waiters := make([]*ffmpeg.Ffmpeg, 0)
	for i := 0; i < len(in); i++ {
		waiters = append(waiters, hapticify(in[i]))
	}
	for i := 0; i < len(waiters); i++ {
		for {
			if !waiters[i].IsRunning() {
				break
			}
		}
	}
}

func hapticify(in string) *ffmpeg.Ffmpeg {
	fmt.Printf("PROCESSING: %s\n", in)

	//Load the input audio
	audio, err := os.ReadFile(in)
	if err != nil {
		panic(err)
	}
	audioBuf := crunchio.NewBuffer(in, audio)

	ff := ffmpeg.NewFFmpeg("libvorbis", "ogg") //Android haptics must be OGG Vorbis
	ff.SetBufferAudioIn(audioBuf)              //Input the loaded audio as a streamable buffer
	ff.SetOnExit(save)                         //Check for errors and save the output once conversion completes
	ff.SetMetadata("ANDROID_HAPTIC", "2")      //Tell Android we have stereo haptics
	ff.SetMetadata("HAPTICIFY", "JoshuaDoes")  //Watermark the final audio :D
	ff.SetInputChannels(2)                     //Mix input to stereo
	ff.SetOutputChannels(4)                    //Mix output to stereo audio + stereo haptics
	ff.SetOutputBitrate(960000)                //960Kbps = 480Kbps audio + 480Kbps haptics, 240Kbps per channel
	ff.SetOutputRate(48000)                    //48KHz audio + haptics
	ff.SetPrecision("f64")                     //Use 64-bit floating point precision for filters and mixing
	ff.SetThreads(runtime.NumCPU())            //Use all CPU cores
	ff.SetBufferLength(time.Millisecond * 20)  //20ms audio buffer for low latency (TODO: not yet implemented!)

	hp := ff.NewFilter()
	hp.SetOutputChannels(0, 1)
	hp.Add(filterhp)
	hp.Add(filtervol)

	lp := ff.NewFilter()
	lp.SetOutputChannels(2, 3)
	lp.Add(filterlp)
	lp.Add(filterhap)

	//Start the conversion process
	if err := ff.Start(); err != nil {
		panic(err)
	}
	return ff
}

func save(ff *ffmpeg.Ffmpeg) {
	in := ff.GetBufferAudioIn().GetName()

	if err := ff.Error(); err != nil {
		fmt.Printf("ERRORS ENCOUNTERED: %s\n%v\n\n", in, err)
	}

	stats := ff.GetBufferStats().Bytes()
	fmt.Printf("STATS: %d bytes\n%s\n\n", len(stats), in)

	audioOut := ff.GetBufferAudioOut().Bytes()
	fmt.Printf("AUDIO OUT: %d bytes\n%s\n\n", len(audioOut), in)

	if len(audioOut) > 0 {
		out := noext(in) + ext
		os.WriteFile(out, audioOut, 0777)
		fmt.Printf("IN: %s\nOUT: %s\n\n", in, out)
	}
}

func files(in string) []string {
	f := make([]string, 0)
	if dir, err := os.ReadDir(in); err == nil {
		for i := 0; i < len(dir); i++ {
			if dir[i].IsDir() {
				continue
			}
			name := dir[i].Name()
			if in != "." {
				name = in + "/" + name
			}
			f = append(f, name)
		}
	} else {
		f = append(f, in)
	}
	return f
}

func noext(fileName string) string {
    if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
        return fileName[:pos]
    }
    return fileName
}
