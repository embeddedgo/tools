Emgo is a thin wrapper over go command. It has two main purposes:

1. Makes it easier to use Embedded Go for embedded programming in presence of orignal Go used for system programming.

2. Allow you to customize some of the Embedded Go environment variables using a `build.cfg` file.

### build.cfg customizable environment variables

#### Go variables (see https://pkg.go.dev/cmd/go#hdr-Environment_variables)

`GOOS` (default: "noos")

`GOARCH` (set implicitly according to `GOTARGET`)

`GOARM` (default value dependent on `GOTARGET`)

#### Embedded Go specific variables

`GOTARGET` (required, no default value)

Specifies the target MCU/SOC family.

Supported values: imxrt1060, k210, n64, noostest, nrf52840, stm32f215, stm32f407, stm32f412, stm32h7x3, stm32l4x6

`GOTEXT` (default value dependent on `GOTARGET`)

Specifies the beggining of code memory, usually Flash. For most targets its default value is infered from `GOTARGET`. The exception is nRF52840 where you must specify it explicitly because of the possibly preprogrammed bootloader and softdevice (set to 0x27000 for bootloader+SD140, 0x1000 for bootloader only, 0 if you don't use any of them).

CAUTION! Wrong or default setting may destroy the preprogrammed bootloader on any target.

`GOMEM` (required, has no default value for most targets)

Ddescribes the structure of available RAM. Currently at most two RAM regions can be specified. The first one is considered DMA capable and available for the user code (stacks, heap, global variables). The second one (if exists) is used only for the runtime internal structures making more of the DMA capable RAM available for the user code.

The format is START_ADDRESS:SIZE or START_ADDRESS1:SIZE1,START_ADDRESS2:SIZE2

`GOOUT` (default: "")

By default `emgo build` produces an ELF file only. If GOOUT is set it also produces one of the supported image formats.

Supported values: bin, hex, z64.

`GOSTRIPFN` (default: 0)

Allows to slightly reduce the size of compiled binary at the cost of less information in the stack traces.

Supported values: 0 - do nothing, 1 - remove package path.

`ISRNAMES` (default: "")

Specifies a package containing the interrupt names to produce a `zisrnames.go` file. This file translates the interrupt handler names based on the interrupt names from specified package to the names known by Embedded Go compiler baesd on the interrupt numbers.

If not set the `emgo` uses a default interrupt package infered from `GOTARGET`. Set `ISRNAMES` to `no` to avoid generation of `zisrnames.go`.

The `zisrnames.go` file is deleted after a successful build. You can see its content if the compilation fails.