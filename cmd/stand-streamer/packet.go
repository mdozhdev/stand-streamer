package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/edgeware/mp4ff/avc"
	"github.com/edgeware/mp4ff/bits"
	"github.com/edgeware/mp4ff/mp4"
	"github.com/nareix/joy5/av"
)

func getSample(sampleNr int, stbl *mp4.StblBox, mdat *mp4.MdatBox) ([]byte, error) {
	chunkNr, sampleNrAtChunkStart, err := stbl.Stsc.ChunkNrFromSampleNr(sampleNr)
	mdatPayloadStart := mdat.PayloadAbsoluteOffset()
	if err != nil {
		return nil, err
	}
	offset := getChunkOffset(stbl, chunkNr)
	for sNr := sampleNrAtChunkStart; sNr < sampleNr; sNr++ {
		offset += int64(stbl.Stsz.GetSampleSize(sNr))
	}
	size := stbl.Stsz.GetSampleSize(sampleNr)
	offsetInMdatData := uint64(offset) - mdatPayloadStart
	sample := mdat.Data[offsetInMdatData : offsetInMdatData+uint64(size)]
	return sample, nil
}

func getDecoderConfigPacket(data []byte) (pkt av.Packet, err error) {
	return av.Packet{
		Type:       av.H264DecoderConfig,
		Data:       data,
		IsKeyFrame: false,
		Time:       0,
		CTime:      0,
	}, nil
}

func getAVPacket(nalu []byte, isKey bool, time time.Duration) (pkt av.Packet, err error) {
	var packetType int

	//We are using hardcoded 4 byte lenght
	naluType := avc.GetNaluType(nalu[0])

	switch naluType {
	case avc.NALU_NON_IDR, avc.NALU_IDR:
		packetType = av.H264
	case avc.NALU_SEI:
		packetType = av.Metadata
		isKey = false
	default:
		return av.Packet{}, fmt.Errorf("Error reading type of the NAL unit")
	}

	data := make([]byte, 4)
	v := uint32(len(nalu))
	binary.BigEndian.PutUint32(data, v)
	data = append(data, nalu...)

	// TODO figure out using CTime
	return av.Packet{
		Type:       packetType,
		Data:       data,
		IsKeyFrame: isKey,
		Time:       time,
		CTime:      time,
	}, nil
}

func findFirstVideoTrak(moov *mp4.MoovBox) (*mp4.TrakBox, error) {
	for _, inTrak := range moov.Traks {
		hdlrType := inTrak.Mdia.Hdlr.HandlerType
		if hdlrType != "vide" {
			continue
		}
		return inTrak, nil
	}
	return nil, fmt.Errorf("No video track found")
}

func getChunkOffset(stbl *mp4.StblBox, chunkNr int) int64 {
	if stbl.Stco != nil {
		return int64(stbl.Stco.ChunkOffset[chunkNr-1])
	}
	if stbl.Co64 != nil {
		return int64(stbl.Co64.ChunkOffset[chunkNr-1])
	}
	panic("Neither stco nor co64 is set")
}

func getDecoderConfig(a avc.DecConfRec) ([]byte, error) {
	sw := bits.NewFixedSliceWriter(int(a.Size()))
	err := a.EncodeSW(sw)
	if err != nil {
		return nil, err
	}

	return sw.Bytes(), err
}
