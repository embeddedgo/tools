// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
)

func nrf5tweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			nfr5common(p)
			switch p.Name {
			case "ficr":
				nrf5ficr(p)
			case "gpio":
				nrf5gpio(p)
			case "nvmc":
				nrf5nvmc(p)
			case "ppi":
				nrf5ppi(p)
			case "rtc":
				nrf5rtc(p)
			case "spi":
				nrf5spi(p)
			case "uicr":
				nrf5uicr(p)
			case "uart", "uarte":
				nrf5uart(p)
			default:
				p.Insts = nil
			}
		}
	}
}

func nfr5common(p *Periph) {
	for _, r := range p.Regs {
		r.Descr = strings.Replace(r.Descr, "Description cluster: ", "", -1)
		r.Descr = strings.Replace(r.Descr, "Description collection: ", "", -1)
		switch r.Name {
		case "SHORTS":
			for _, b := range r.Bits {
				b.Values = nil
			}
		case "INTEN", "INTENSET", "INTENCLR", "EVTEN", "EVTENSET",
			"EVTENCLR":
			r.Bits = nil
		case "ERRORSRC":
			for _, b := range r.Bits {
				b.Name = "E" + b.Name
			}
		case "ENABLE":
			for _, b := range r.Bits {
				if b.Name == "ENABLE" {
					b.Name = "EN"
				}
			}
		}
		switch {
		case strings.HasPrefix(r.Name, "TASKS_"):
			r.Name = "TASK_" + r.Name[6:]
			r.Bits = nil
		case strings.HasPrefix(r.Name, "EVENTS_"):
			r.Name = "EVENT_" + r.Name[7:]
			r.Bits = nil
		case strings.HasPrefix(r.Name, "PSEL_"):
			r.Bits = nil
		}
		if len(r.Bits) == 1 {
			if b := r.Bits[0]; b.Mask == 0xFFFFFFFF && len(b.Values) <= 1 {
				r.Bits = nil
			}
		}
		for _, b := range r.Bits {
			if b.Mask == 1 {
				b.Values = nil
			}
		}
	}
}

func nrf5ficr(p *Periph) {
	var a, b, t *Reg
	for i, r := range p.Regs {
		switch r.Name {
		case "INFO_VARIANT", "INFO_PACKAGE", "INFO_RAM", "INFO_FLASH":
			for _, b := range r.Bits {
				for _, v := range b.Values {
					if v.Name == "Unspecified" {
						v.Name = b.Name[:1] + "unspec"
					} else if v.Name[0] == 'K' {
						v.Name = b.Name[:1] + v.Name[1:] + "K"
					}
				}
			}
		case "DEVICEADDRTYPE":
			for _, b := range r.Bits {
				if b.Name == "DEVICEADDRTYPE" {
					b.Name = "PUBLIC"
				}
			}
		case "PRODTEST":
			for _, b := range r.Bits {
				if b.Name == "PRODTEST" {
					b.Name = "PTEST"
				}
			}
		}
		switch {
		case r.Name == "TEMP_A0":
			a = r
			a.Name = "TEMP_A"
			a.Len = 1
			a.Descr = "Slope definition"
		case strings.HasPrefix(r.Name, "TEMP_A"):
			a.Len++
			p.Regs[i] = nil
		case r.Name == "TEMP_B0":
			b = r
			b.Name = "TEMP_B"
			b.Len = 1
			b.Descr = "Y-intercept"
		case strings.HasPrefix(r.Name, "TEMP_B"):
			b.Len++
			p.Regs[i] = nil
		case r.Name == "TEMP_T0":
			t = r
			t.Name = "TEMP_T"
			t.Len = 1
			t.Descr = "Segment end"
		case strings.HasPrefix(r.Name, "TEMP_T"):
			t.Len++
			p.Regs[i] = nil
		}
	}
}

func nrf5gpio(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "OUT", "OUTSET", "OUTCLR", "IN", "DIR", "DIRSET", "DIRCLR", "LATCH":
			r.Bits = nil
		case "DETECTMODE":
			for _, b := range r.Bits {
				if b.Name == "DETECTMODE" {
					b.Name = "DETECT"
					for _, v := range b.Values {
						switch v.Name {
						case "Default":
							v.Name = "Direct"
						case "LDETECT":
							v.Name = "Latched"
						}
					}
				}
			}
		case "PIN_CNF":
			for _, b := range r.Bits {
				switch b.Name {
				case "DIR":
					b.Name = "DIRECTION"
				case "PULL":
					for _, v := range b.Values {
						if v.Name == "Disabled" {
							v.Name = "NoPull"
						}
					}
				}
			}
		}
	}
}

func nrf5nvmc(p *Periph) {
	for i, r := range p.Regs {
		switch r.Name {
		case "READY", "READYNEXT", "ERASEALL", "ERASEUICR":
			r.Bits = nil
		case "ICACHECNF":
			for _, b := range r.Bits {
				b.Values = nil
			}
		case "ERASEPCR1":
			p.Regs[i] = nil
		}
	}
}

func nrf5ppi(p *Periph) {
	for _, r := range p.Regs {
		r.Bits = nil
		if r.Name == "FORK" && len(r.SubRegs) == 1 {
			r.Name += "_" + r.SubRegs[0].Name
			r.SubRegs = nil
		}
		for _, sr := range r.SubRegs {
			sr.Bits = nil
		}
	}
}

func nrf5rtc(p *Periph) {
	for _, r := range p.Regs {
		r.Bits = nil
	}
}

func nrf5spi(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "RXD", "TXD":
			r.Bits = nil
		case "FREQUENCY":
			r.Bits[0].Name = "FREQ"
		}
	}
}

func nrf5uart(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "BAUDRATE":
			for _, b := range r.Bits {
				if b.Name == "BAUDRATE" {
					b.Name = "BR"
				}
			}
		case "CONFIG":
			for _, b := range r.Bits {
				if b.Name == "HFC" {
					for _, v := range b.Values {
						switch v.Name {
						case "Disabled":
							v.Name = "None"
						case "Enabled":
							v.Name = "RTSCTS"
						}
					}
				}
			}
		case "RXD", "TXD", "RXD_MAXCNT", "RXD_AMOUNT", "TXD_MAXCNT",
			"TXD_AMOUNT":
			r.Bits = nil
		}
	}
}

func nrf5uicr(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "NRFFW", "NRFHW", "CUSTOMER":
			r.Bits = nil
		case "NFCPINS":
			for _, b := range r.Bits {
				for _, v := range b.Values {
					if v.Name == "Disabled" {
						v.Name = "GPIO"
					}
				}
			}
		case "DEBUGCTRL":
			for _, b := range r.Bits {
				switch b.Name {
				case "CPUNIDEN":
					for _, v := range b.Values {
						v.Name = "ID" + v.Name
					}
				case "CPUFPBEN":
					for _, v := range b.Values {
						v.Name = "FPB" + v.Name
					}
				}
			}
		case "REGOUT0":
			for _, b := range r.Bits {
				for _, v := range b.Values {
					switch {
					case v.Name == "DEFAULT":
						v.Name = "Vdefault"
					case len(v.Name) == 3 && v.Name[1] == 'V':
						v.Name = "V" + v.Name[0:1] + "_" + v.Name[2:3]
					}
				}
			}
		}
	}
}
