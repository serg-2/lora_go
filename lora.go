package main

/*
#cgo CFLAGS: -I .
#cgo LDFLAGS: -L . -lmainlib
#include <wiringPi.h>
#include <wiringPiSPI.h>
*/
import "C"
import "fmt"
import "os"
import "unsafe"
import "time"

const go_REG_OPMODE=0x01
const go_OPMODE_MASK=0x07
const go_not_OPMODE_MASK=0xF8
const go_OPMODE_SLEEP=0x00
const go_freq=868100000
const go_REG_FRF_MSB=0x06
const go_REG_FRF_MID=0x07
const go_REG_FRF_LSB=0x08
const go_REG_SYNC_WORD=0x39
const go_REG_MODEM_CONFIG=0x1D
const go_REG_MODEM_CONFIG2=0x1E
const go_REG_MODEM_CONFIG3=0x26
const go_REG_SYMB_TIMEOUT_LSB=0x1F

const go_REG_VERSION=0x42

const go_REG_MAX_PAYLOAD_LENGTH=0x23
const go_REG_PAYLOAD_LENGTH=0x22
const go_PAYLOAD_LENGTH=0x40
const go_REG_HOP_PERIOD=0x24
const go_REG_FIFO_ADDR_PTR=0x0D

const go_REG_FIFO_RX_BASE_AD=0x0F
const go_REG_LNA=0x0C
const go_LNA_MAX_GAIN=0x23
const go_REG_FIFO=0x00

const go_OPMODE_LORA=0x80
const go_OPMODE_STANDBY=0x01

const go_OPMODE_TX=0x03

const go_RegPaRamp=0x0A
const go_RegPaConfig=0x09
const go_RegPaDac=0x5A

const go_RegDioMapping1=0x40
const go_MAP_DIO0_LORA_TXDONE=0x40
const go_MAP_DIO1_LORA_NOP=0x30
const go_MAP_DIO2_LORA_NOP=0xC0
const go_REG_IRQ_FLAGS=0x12
const go_REG_IRQ_FLAGS_MASK=0x11
const go_IRQ_LORA_TXDONE_MASK=0x08
const go_not_IRQ_LORA_TXDONE_MASK=0xF7
const go_REG_FIFO_TX_BASE_AD=0x0E

const (
  SF7 = 7 
  SF8 = 8
  SF9 = 9
  SF10 = 10
  SF11 = 11
  SF12 = 12
)

const go_sf=SF7

var sx1272 bool
var go_CHANNEL int = 0
var go_ssPin int = 6
var go_dio0 int = 7
var go_RST int = 0

func go_selectreceiver() {
  C.digitalWrite(C.int(go_ssPin), C.LOW)
}

func go_unselectreceiver() {
  C.digitalWrite(C.int(go_ssPin), C.HIGH)
}

func go_writeReg(addr byte, value byte) {
  spibuf:= [2]C.uchar{}
  spibuf[0] = C.uchar( addr | 0x80 )
  spibuf[1] = C.uchar( value )
  go_selectreceiver()
  spibufPtr:= (*C.uchar)(unsafe.Pointer(&spibuf))
  C.wiringPiSPIDataRW(C.int(go_CHANNEL), spibufPtr, 2)
  go_unselectreceiver()
}

func go_readReg(addr byte) byte {
  spibuf:= [2]C.uchar{}
  go_selectreceiver()
  spibuf[0] = C.uchar( addr & 0x7F )
  spibuf[1] = C.uchar( 0x00 )
  spibufPtr:= (*C.uchar)(unsafe.Pointer(&spibuf))
  C.wiringPiSPIDataRW(C.int(go_CHANNEL), spibufPtr, 2)
  go_unselectreceiver()
  return byte (spibuf[1]) 
}

func go_opmode(mode byte) {
  go_writeReg(go_REG_OPMODE, go_readReg (go_REG_OPMODE) & go_not_OPMODE_MASK | mode )
}

func go_opmodeLora() {
  var u byte = go_OPMODE_LORA
  if sx1272 == false {
    u|= 0x8   // TBD: sx1276 high freq  
  }
  go_writeReg(go_REG_OPMODE, u)
}


func go_SetupLoRa() {
  C.digitalWrite(C.int(go_RST), C.HIGH)
  time.Sleep(100*time.Millisecond)
  C.digitalWrite(C.int(go_RST), C.LOW)
  time.Sleep(100*time.Millisecond)

  var version byte = go_readReg(go_REG_VERSION)

  if version == 0x22 {
    // sx1272
    fmt.Println("SX1272 detected, starting.")
    sx1272 = true
  } else {
    // sx1276?
    C.digitalWrite(C.int(go_RST), C.LOW)
    time.Sleep(100*time.Millisecond)
    C.digitalWrite(C.int(go_RST), C.HIGH)
    time.Sleep(100*time.Millisecond)
    version = go_readReg(go_REG_VERSION)
    if version == 0x12 {
      // sx1276
      fmt.Println("SX1276 detected, Starting.")
      sx1272= false
    } else {
      fmt.Println("Unrecognized transceiver.")
      //fmt.Printf("Transceiver version %x",version)
      os.Exit(1)
    }
  }
  
  go_opmode(go_OPMODE_SLEEP)
  
  //set frequency
  var frf uint64 = uint64( go_freq << 19 ) / 32000000
  go_writeReg(go_REG_FRF_MSB, byte(frf >> 16))
  go_writeReg(go_REG_FRF_MID, byte(frf >> 8))
  go_writeReg(go_REG_FRF_LSB, byte(frf >> 0))

  go_writeReg(go_REG_SYNC_WORD, 0x34) //LoRaWAN public sync word

  if sx1272 {
     if go_sf == SF11 || go_sf == SF12 {
        go_writeReg(go_REG_MODEM_CONFIG,0x0B);
     } else {
        go_writeReg(go_REG_MODEM_CONFIG,0x0A);
     }
     go_writeReg(go_REG_MODEM_CONFIG2,(go_sf<<4) | 0x04);
  } else {
     if go_sf == SF11 || go_sf == SF12 {
        go_writeReg(go_REG_MODEM_CONFIG3,0x0C);
     } else {
        go_writeReg(go_REG_MODEM_CONFIG3,0x04);
     }
     go_writeReg(go_REG_MODEM_CONFIG,0x72);
     go_writeReg(go_REG_MODEM_CONFIG2,(go_sf<<4) | 0x04);
  }
  
   if go_sf == SF10 || go_sf == SF11 || go_sf == SF12 {
     go_writeReg(go_REG_SYMB_TIMEOUT_LSB,0x05);
   } else {
     go_writeReg(go_REG_SYMB_TIMEOUT_LSB,0x08);
   }
   go_writeReg(go_REG_MAX_PAYLOAD_LENGTH,0x80);
   go_writeReg(go_REG_PAYLOAD_LENGTH,go_PAYLOAD_LENGTH);
   go_writeReg(go_REG_HOP_PERIOD,0xFF);
   go_writeReg(go_REG_FIFO_ADDR_PTR, go_readReg(go_REG_FIFO_RX_BASE_AD));

   go_writeReg(go_REG_LNA, go_LNA_MAX_GAIN);
}

func go_configPower (pw int8) {
  if sx1272 == false {
    // no boost used for now
    if pw >=17 {
      pw=15
    } else if pw < 2 {
      pw=2
    }
    // check board type for BOOST pin
    go_writeReg(go_RegPaConfig, byte( 0x80 | byte(pw & 0xf )))
    go_writeReg(go_RegPaDac, go_readReg(go_RegPaDac)|0x4)
  } else {
    // set PA config (2-17 dBm using PA_BOOST)
    if pw > 17 {
      pw = 17
    } else if pw < 2 {
      pw = 2
    }
    go_writeReg(go_RegPaConfig, byte(0x80|byte(pw-2)))
  }
}

func go_txlora (send_string string) {
    // set the IRQ mapping DIO0=TxDone DIO1=NOP DIO2=NOP
    go_writeReg(go_RegDioMapping1, go_MAP_DIO0_LORA_TXDONE|go_MAP_DIO1_LORA_NOP|go_MAP_DIO2_LORA_NOP)
    // clear all radio IRQ flags
    go_writeReg(go_REG_IRQ_FLAGS, 0xFF)
    // mask all IRQs but TxDone
    go_writeReg(go_REG_IRQ_FLAGS_MASK, go_not_IRQ_LORA_TXDONE_MASK)

    // initialize the payload size and address pointers
    go_writeReg(go_REG_FIFO_TX_BASE_AD, 0x00)
    go_writeReg(go_REG_FIFO_ADDR_PTR, 0x00)
    go_writeReg(go_REG_PAYLOAD_LENGTH, byte(len(send_string)))

    // download buffer to the radio FIFO
    go_writeBuf(go_REG_FIFO, send_string)
    // now we actually start the transmission
    go_opmode(go_OPMODE_TX)

    fmt.Printf("send: %s\n", send_string)

}

func go_writeBuf (addr byte, send_string string) {
    var string_by_byte []byte=[]byte(send_string)
    spibuf:= [256]C.uchar{}                                                  
    spibuf[0] = C.uchar(addr | 0x80)
    for i:= 0; i < len(send_string); i++ {
        spibuf[i + 1] = C.uchar(string_by_byte[i])
    }                                                               
    go_selectreceiver()
    spibufPtr:= (*C.uchar)(unsafe.Pointer(&spibuf))              
    C.wiringPiSPIDataRW(C.int(go_CHANNEL), spibufPtr, C.int(len(send_string)+1))      
    go_unselectreceiver()                  
}

func main() {
  if len(os.Args[1:]) == 0 {
    fmt.Printf("Usage: %v sender|rec [message]\n",os.Args[0])
    os.Exit(0)
  }
  
  C.wiringPiSetup()
  C.pinMode(C.int(go_ssPin), C.OUTPUT)
  C.pinMode(C.int(go_dio0), C.INPUT)
  C.pinMode(C.int(go_RST), C.OUTPUT)
  C.wiringPiSPISetup(C.int(go_CHANNEL),500000)

  go_SetupLoRa()

// Starting Send
  go_opmodeLora()
  go_opmode(go_OPMODE_STANDBY)

  go_writeReg(go_RegPaRamp, (go_readReg(go_RegPaRamp) & 0xF0) | 0x08); // set PA ramp-up time 50 uSec

  go_configPower(23)
  
  fmt.Printf("Send packets at SF%d on %f Mhz.\n", go_sf, float64(float64(go_freq) / 1000000) )
  fmt.Println("-----------------------")

  var string_to_send string="test23"

  for {
    go_txlora(string_to_send)
    time.Sleep(2*time.Second)  
  }

}

