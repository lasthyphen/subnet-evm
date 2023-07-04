// (c) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"github.com/lasthyphen/dijetalgo/codec"
	"github.com/lasthyphen/dijetalgo/codec/linearcodec"
	"github.com/lasthyphen/dijetalgo/codec/reflectcodec"
	"github.com/lasthyphen/dijetalgo/utils/units"
	"github.com/lasthyphen/dijetalgo/utils/wrappers"
)

const (
	codecVersion   uint16 = 0
	maxMessageSize        = 512 * units.KiB
	maxSliceLen           = maxMessageSize
)

// Codec does serialization and deserialization
var c codec.Manager

func init() {
	c = codec.NewManager(maxMessageSize)
	lc := linearcodec.New(reflectcodec.DefaultTagName, maxSliceLen)

	errs := wrappers.Errs{}
	errs.Add(
		lc.RegisterType(&Txs{}),
		c.RegisterCodec(codecVersion, lc),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
