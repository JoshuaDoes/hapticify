package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var (
	ErrorAlreadyRunning error = fmt.Errorf("ffmpeg: already running")
)

func main() {
	ff := NewFFmpeg("libvorbis", "ogg")         //Android haptics must be OGG Vorbis
	ff.SetInput(os.Args[1])                     //Input file
	ff.SetOutput(os.Args[2] + ".hapticify.ogg") //Output file
	ff.SetMetadata("ANDROID_HAPTIC", "2")       //Tell Android we have stereo haptics
	ff.SetInputChannels(2)                      //Mix input to stereo
	ff.SetOutputChannels(4)                     //Mix output to stereo audio + stereo haptics
	ff.SetOutputBitrate(960000)                 //960Kbps = 480Kbps audio + 480Kbps haptics, 240Kbps per channel
	ff.SetOutputRate(48000)                     //48KHz audio + haptics
	ff.SetPrecision("f64")                      //Use 64-bit floating point precision for filters and mixing
	ff.SetThreads(runtime.NumCPU())             //Use all CPU cores
	ff.SetBufferLength(time.Millisecond * 20)   //20ms audio buffer for low latency

	hp := ff.NewFilter()
	hp.SetInputChannels(0, 1)
	hp.SetOutputChannels(0, 1)
	hp.Add(NewFilterHighpass(70))

	lp := ff.NewFilter()
	lp.SetInputChannels(0, 1)
	lp.SetOutputChannels(2, 3)
	lp.Add(NewFilterLowpass(150))
	lp.Add(NewFilterEqualizer(10, 20, 5.0))
	lp.Add(NewFilterEqualizer(30, 20, 4.5))
	lp.Add(NewFilterEqualizer(50, 20, 4.0))
	lp.Add(NewFilterEqualizer(65, 10, 3.0))
	lp.Add(NewFilterEqualizer(110, 80, 2.0))

	//Start the conversion process
	if err := ff.Start(); err != nil {
		panic(err)
	}

	//Wait until it finishes
	ff.Wait()
}
