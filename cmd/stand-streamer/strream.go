package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/edgeware/mp4ff/avc"
	"github.com/edgeware/mp4ff/mp4"
	"github.com/nareix/joy5/av"
	"github.com/nareix/joy5/av/pktop"
	"github.com/nareix/joy5/format"
)

func stream(name, src, baseUrl string, wg *sync.WaitGroup) {
	defer wg.Done()
	var err error
	dst := baseUrl + "/" + name
	fmt.Println(name, ": DST ", dst)

	//rtmp stuff
	foW := &format.URLOpener{}
	var fw *format.Writer
	var re *pktop.NativeRateLimiter

	if fw, err = foW.Create(dst); err != nil {
		fmt.Println(name, ": Error creating dst: ", err)
		return
	}

	// file stuff
	file, err := os.Open(src)
	if err != nil {
		fmt.Println(name, ": Error opening src: ", err)
		return
	}
	defer file.Close()

	parsedMp4, err := mp4.DecodeFile(file)
	if err != nil {
		fmt.Println(name, ": Error reading src: ", err)
		return
	}

	videoTrak, err := findFirstVideoTrak(parsedMp4.Moov)
	if err != nil {
		fmt.Println(name, ": Error finding first trak: ", err)
		return
	}

	stbl := videoTrak.Mdia.Minf.Stbl
	stss := stbl.Stss
	nrSamples := stbl.Stsz.SampleNumber
	mdat := parsedMp4.Mdat
	// mdatPayloadStart := mdat.PayloadAbsoluteOffset()

	var codec string

	if stbl.Stsd.AvcX != nil {
		codec = "avc"
	} else if stbl.Stsd.HvcX != nil {
		codec = "hevc"
		fmt.Println(name, ": HEVC Codec is not supported yet. Exiting.")
		return
	}

	var decConf []byte
	switch codec {
	case "avc":
		decConf, err = getDecoderConfig(stbl.Stsd.AvcX.AvcC.DecConfRec)
		if err != nil {
			fmt.Println(name, ": Error getting DecoderConfig: ", err)
			return
		}
	case "hevc":
		// HEVC is not implemented
		if stbl.Stsd.HvcX.Type() == "hev1" {
			fmt.Printf("Warning: there should be no parameter sets in sample descriptor hev1\n")
		}
		// vpsNalus := stsd.HvcX.HvcC.GetNalusForType(hevc.NALU_VPS)
		// spsNalus := stsd.HvcX.HvcC.GetNalusForType(hevc.NALU_SPS)
		// ppsNalus := stsd.HvcX.HvcC.GetNalusForType(hevc.NALU_PPS)
	}

	pkt, err := getDecoderConfigPacket(decConf)
	if err != nil {
		fmt.Println(name, ": Error getting DecoderConfig packet: ", err)
		return
	}

	if err = fw.WritePacket(pkt); err != nil {
		fmt.Println(name, ": Error writing DecoderConfig packet: ", err)
		return
	}

	for sampleNr := 1; sampleNr <= int(nrSamples); sampleNr++ {
		sample, err := getSample(sampleNr, stbl, mdat)
		if err != nil {
			fmt.Println(name, ": Error getting sample: ", err)
			return
		}

		//TODO figure out proper time units
		decTime, _ := stbl.Stts.GetDecodeTime(uint32(sampleNr))
		rDecTime := time.Millisecond * time.Duration(decTime) / 10

		//TODO Figure out cTime usage
		//var cto int32 = 0
		// if stbl.Ctts != nil {
		// 	cto = stbl.Ctts.GetCompositionTimeOffset(uint32(sampleNr))
		// }

		nalus, err := avc.GetNalusFromSample(sample)
		for _, nalu := range nalus {
			isKey := stss.IsSyncSample(uint32(sampleNr))

			pkt, err = getAVPacket(nalu, isKey, rDecTime)
			if err = fw.WritePacket(pkt); err != nil {
				fmt.Println(name, ": Error gettting packet: ", err)
				return
			}

			if re == nil {
				re = pktop.NewNativeRateLimiter()
			}

			re.Do([]av.Packet{pkt})

			fmt.Printf("%s: %s\n", name, pkt.String())
			if err = fw.WritePacket(pkt); err != nil {
				fmt.Println(name, ": Error writing packet: ", err)
				return
			}
		}
	}
	return
}
