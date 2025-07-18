// Copyright 2023 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imxmbr

import (
	"bytes"
	"encoding/binary"
)

func Make(flashSize, imageSize int, flexRAMCfg uint32) []byte {
	if flashSize < 0 || imageSize < 0 {
		panic("flashSize<0 || imageSize<0")
	}
	const mbrSize = mbrEndAddr - baseAddr
	if imageSize == 0 {
		imageSize = flashSize - mbrSize
	}

	flashConfig.MemCfg.SFlashA1Size = uint32(flashSize)
	buf := bytes.NewBuffer(make([]byte, 0, mbrSize))

	binary.Write(buf, binary.LittleEndian, flashConfig)
	for a := baseAddr + flashConfigSize; a < ivtAddr; a++ {
		buf.WriteByte(0xff)
	}
	if flexRAMCfg == 0 {
		bootData.Length = uint32(imageSize)
		binary.Write(buf, binary.LittleEndian, regularIVT)
		binary.Write(buf, binary.LittleEndian, bootData)
		for a := bootDataAddr + bootDataSize; a < mbrEndAddr; a++ {
			buf.WriteByte(0xff)
		}
	} else {
		bootData.Length = uint32(pluginAddr + len(plugin)*2 - baseAddr)
		bootData.Plugin = 1
		imageSize -= stage2IVTAddr - baseAddr
		pluginImageSize[0] = uint16(imageSize)
		pluginImageSize[1] = uint16(imageSize >> 16)
		pluginFlexRAMCfg[0] = uint16(flexRAMCfg)
		pluginFlexRAMCfg[1] = uint16(flexRAMCfg >> 16)
		binary.Write(buf, binary.LittleEndian, pluginIVT)
		binary.Write(buf, binary.LittleEndian, bootData)
		for a := bootDataAddr + bootDataSize; a < pluginAddr; a++ {
			buf.WriteByte(0xff)
		}
		binary.Write(buf, binary.LittleEndian, plugin)
		for a := pluginAddr + len(plugin)*2; a < stage2IVTAddr; a++ {
			buf.WriteByte(0xff)
		}
		binary.Write(buf, binary.LittleEndian, stage2IVT)
		for a := stage2IVTAddr + ivtSize; a < mbrEndAddr; a++ {
			buf.WriteByte(0xff)
		}
	}
	return buf.Bytes()
}
