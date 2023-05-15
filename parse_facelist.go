package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type faces struct {
	ngb  string
	rtt  float64
	thg  float64
	tkn  string
	n_oi uint64
	n_in uint64
}

var facelist map[uint64]faces

func parse_facelist(raw []byte) {
	facelist = make(map[uint64]faces)
	var (
		nextface uint64
		length   uint64
		pointer  uint64
		faceid   uint64
		innack   uint64
		outi     uint64
	)

	pointer = 0
	for pointer != uint64(len(raw)) {
		// check per face
		if data := hex.EncodeToString([]byte{raw[pointer]}); data == "80" {
			pointer++
			// fmt.Println("data:", data)
			octet := check_type([]byte{raw[pointer]})
			if octet == 1 {
				length = check_length([]byte{raw[pointer]})
			} else {
				length = check_length(raw[pointer : pointer+octet])
			}
			pointer += octet
			nextface = pointer + length

			for pointer < nextface {
				if data := hex.EncodeToString([]byte{raw[pointer]}); data == "69" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					faceid = get_data(raw[pointer : pointer+length])
					// fmt.Println("faceid: ", faceid)
					pointer += length
				} else if data := hex.EncodeToString([]byte{raw[pointer]}); data == "92" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					outi = get_data(raw[pointer : pointer+length])
					// fmt.Println("outi: ", outi)
					pointer += length
				} else if data := hex.EncodeToString([]byte{raw[pointer]}); data == "97" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					innack = get_data(raw[pointer : pointer+length])
					// fmt.Println("innack: ", innack)
					pointer += length
				} else {
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet + length
				}
			}

			token := make([]byte, 16)
			rand.Read(token)
			r_token := hex.EncodeToString(token)
			fmt.Println(faceid)
			if _, ok := facelist[faceid]; ok {
				facelist[faceid] = faces{n_oi: outi, n_in: innack, tkn: facelist[faceid].tkn, ngb: facelist[faceid].ngb, rtt: facelist[faceid].rtt, thg: facelist[faceid].thg}
			} else {
				facelist[faceid] = faces{n_oi: outi, n_in: innack, tkn: r_token}
			}

		}
	}
}

func check_type(wire []byte) (res uint64) {
	tlv_type := int(wire[0])
	if tlv_type < 253 {
		res = 1
	} else if tlv_type == 253 {
		res = 3
	} else if tlv_type == 254 {
		res = 5
	} else {
		res = 9
	}

	return res
}

func check_length(wire []byte) (res uint64) {
	byte_length := len(wire)
	if byte_length == 1 {
		res = uint64(wire[0])
	} else if byte_length == 2 {
		res = uint64(binary.BigEndian.Uint16(wire))
	} else if byte_length == 4 {
		res = uint64(binary.BigEndian.Uint32(wire))
	} else {
		res = binary.BigEndian.Uint64(wire)
	}

	return res
}

func get_data(wire []byte) (res uint64) {
	byte_length := len(wire)
	if byte_length == 1 {
		res = uint64(wire[0])
	} else if byte_length == 2 {
		res = uint64(binary.BigEndian.Uint16(wire))
	} else if byte_length == 4 {
		res = uint64(binary.BigEndian.Uint32(wire))
	} else {
		res = binary.BigEndian.Uint64(wire)
	}

	return res
}
