// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svd

import (
	"encoding/xml"
	"errors"
	"strconv"
)

type Int int

func (i *Int) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	v, err := strconv.ParseInt(s, 0, 0)
	*i = Int(v)
	return err
}

type Uint uint

func (u *Uint) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	v, err := strconv.ParseUint(s, 0, 0)
	*u = Uint(v)
	return err
}

type Uint64 uint64

func (u *Uint64) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	v, err := strconv.ParseUint(s, 0, 64)
	*u = Uint64(v)
	return err
}

type Device struct {
	Vendor                  *string `xml:"vendor"`
	VendorID                *string `xml:"vendorID"`
	Name                    string  `xml:"name"`
	Series                  *string `xml:"series"`
	Version                 string  `xml:"version"`
	Description             string  `xml:"description"`
	LicenseText             *string `xml:"licenseText"`
	CPU                     *CPU    `xml:"cpu"`
	HeaderSystemFilename    *string `xml:"headerSystemFilename"`
	HeaderDefinitionsPrefix *string `xml:"headerDefinitionsPrefix"`
	AddressUnitBits         Uint    `xml:"addressUnitBits"`
	Width                   Uint    `xml:"width"`
	*RegisterPropertiesGroup
	Peripherals []*Peripheral `xml:"peripherals>peripheral"`
}

type CPU struct {
	Name                string `xml:"name"`
	Revision            string `xml:"revision"`
	Endian              string `xml:"endian"`
	MPUPresent          bool   `xml:"mpuPresent"`
	FPUPresent          bool   `xml:"fpuPresent"`
	FPUDP               *bool  `xml:"fpuDP"`
	DSPPresent          *bool  `xml:"dspPresent"`
	IcachePresent       *bool  `xml:"icachePresent"`
	DcachePresent       *bool  `xml:"dcachePresent"`
	ITCMPresent         *bool  `xml:"itcmPresent"`
	DTCMPresent         *bool  `xml:"dtcmPresent"`
	VTORPresent         *bool  `xml:"vtorPresent"`
	NVICPrioBits        Uint   `xml:"nvicPrioBits"`
	VendorSystickConfig bool   `xml:"vendorSystickConfig"`
	DeviceNumInterrupts *Uint  `xml:"deviceNumInterrupts"`
	SAUNumRegions       *Uint  `xml:"sauNumRegions"`
	//SAURegionsConfig *SAURegionsConfig `xml:"sauNumRegions"`
}

type RegisterPropertiesGroup struct {
	Size       *Uint   `xml:"size"`
	Access     *string `xml:"access"`
	Protection *string `xml:"protection"`
	ResetValue *Uint64 `xml:"resetValue"`
	ResetMask  *Uint64 `xml:"resetMask"`
}

type Peripheral struct {
	DerivedFrom *string `xml:"derivedFrom,attr"`
	*DimElementGroup
	Name                string  `xml:"name"`
	Version             *string `xml:"version"`
	Description         *string `xml:"description"`
	AlternatePeripheral *string `xml:"alternatePeripheral"`
	GroupName           *string `xml:"groupName"`
	PrependToName       *string `xml:"prependToName"`
	AppendToName        *string `xml:"appendToName"`
	HeaderStructName    *string `xml:"headerStructName"`
	DisableCondition    *string `xml:"disableCondition"`
	BaseAddress         Uint64  `xml:"baseAddress"`
	*RegisterPropertiesGroup
	AddressBlock []*AddressBlock `xml:"addressBlock"`
	Interrupts   []*Interrupt    `xml:"interrupt"`
	Registers    []*Register     `xml:"registers>register"`
	Clusters     []*Cluster      `xml:"registers>cluster"`
}

type DimElementGroup struct {
	Dim          Uint    `xml:"dim"`
	DimIncrement Uint    `xml:"dimIncrement"`
	DimIndex     *string `xml:"dimIndex"`
	DimName      *string `xml:"dimName"`
	//DimArrayIndex *DimArrayIndex `xml:"dimArrayIndex"`
}

type AddressBlock struct {
	Offset     Uint64  `xml:"offset"`
	Size       Uint64  `xml:"size"`
	Usage      string  `xml:"usage"`
	Protection *string `xml:"protection"`
}

type Interrupt struct {
	Name        string  `xml:"name"`
	Description *string `xml:"description"`
	Value       Int     `xml:"value"`
}

type Register struct {
	DerivedFrom *string `xml:"derivedFrom,attr"`
	DimElementGroup
	Name              string  `xml:"name"`
	DisplayName       *string `xml:"displayName"`
	Description       *string `xml:"description"`
	AlternateGroup    *string `xml:"alternateGroup"`
	AlternateRegister *string `xml:"alternateRegister"`
	AddressOffset     Uint64  `xml:"addressOffset"`
	*RegisterPropertiesGroup
	DataType            *string          `xml:"dataType"`
	ModifiedWriteValues *string          `xml:"modifiedWriteValues"`
	WriteConstraint     *WriteConstraint `xml:"writeConstraint"`
	ReadAction          *string          `xml:"readAction"`
	Fields              []*Field         `xml:"fields>field"`
}

type WriteConstraint struct {
	WriteAsRead         *bool  `xml:"writeAsRead"`
	UseEnumeratedValues *bool  `xml:"useEnumeratedValues"`
	Range               *Range `xml:"range"`
}

type Range struct {
	Minimum Uint64 `xml:"minimum"`
	Maximum Uint64 `xml:"maximum"`
}

type Field struct {
	DerivedFrom *string `xml:"derivedFrom,attr"`
	DimElementGroup
	Name        string  `xml:"name"`
	Description *string `xml:"description"`
	*BitRangeOffsetWidth
	*BitRangeLSBMSB
	BitRangePattern     *string             `xml:"bitRange"`
	Access              *string             `xml:"access"`
	ModifiedWriteValues *string             `xml:"modifiedWriteValues"`
	WriteConstraint     *WriteConstraint    `xml:"writeConstraint"`
	ReadAction          *string             `xml:"readAction"`
	EnumeratedValues    []*EnumeratedValues `xml:"enumeratedValues"`
}

type BitRangeOffsetWidth struct {
	BitOffset Uint  `xml:"bitOffset"`
	BitWidth  *Uint `xml:"bitWidth"`
}

type BitRangeLSBMSB struct {
	LSB Uint `xml:"lsb"`
	MSB Uint `xml:"msb"`
}

type EnumeratedValues struct {
	DerivedFrom     *string            `xml:"derivedFrom,attr"`
	Name            *string            `xml:"name"`
	HeaderEnumName  *string            `xml:"headerEnumName"`
	Usage           *string            `xml:"usage"`
	EnumeratedValue []*EnumeratedValue `xml:"enumeratedValue"`
}

type EnumeratedValue struct {
	Name        *string `xml:"name"`
	Description *string `xml:"description"`
	Value       *string `xml:"value"`
	IsDefault   *bool   `xml:"isDefault"`
}

var ErrNilValue = errors.New("nil value")

func (ev *EnumeratedValue) Val() (uint64, error) {
	if ev.Value == nil {
		return 0, ErrNilValue
	}
	s := *ev.Value
	if s[0] == '#' {
		// binary #1011 or binary #1x0x "do not care" format
		a := make([]byte, len(s)+1)
		a[0] = '0'
		a[1] = 'b'
		for i := 1; i < len(s); i++ {
			b := s[i]
			if b == 'x' {
				b = '0'
			}
			a[i+1] = b
		}
		s = string(a)
	}
	return strconv.ParseUint(s, 0, 64)
}

type Cluster struct {
	DerivedFrom *string `xml:"derivedFrom,attr"`
	DimElementGroup
	Name             string  `xml:"name"`
	Description      *string `xml:"description"`
	AlternateCluster *string `xml:"alternateCluster"`
	HeaderStructName *string `xml:"headerStructName"`
	AddressOffset    Uint64  `xml:"addressOffset"`
	*RegisterPropertiesGroup
	Registers []*Register `xml:"register"`
	Clusters  []*Cluster  `xml:"cluster"`
}
