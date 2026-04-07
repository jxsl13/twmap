package twmap

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"io"
	"slices"

	"github.com/klauspost/compress/zlib"
)

// Write serialises the Map into the Teeworlds datafile (v4) format,
// writing the result to w. The output can be loaded by TW/DDNet clients.
func (m *Map) Write(w io.Writer) error {
	b := newDatafileBuilder()

	// ── version ──────────────────────────────────────────────────────────
	b.addItem(mapItemTypeVersion, 0, []int32{1})

	// ── info ─────────────────────────────────────────────────────────────
	m.writeInfo(b)

	// ── images ───────────────────────────────────────────────────────────
	m.writeImages(b)

	// ── envelopes + env points ───────────────────────────────────────────
	m.writeEnvelopes(b)

	// ── sounds ───────────────────────────────────────────────────────────
	m.writeSounds(b)

	// ── groups & layers ──────────────────────────────────────────────────
	m.writeGroupsAndLayers(b)

	return b.finish(w)
}

// ── datafile builder ─────────────────────────────────────────────────────────

type datafileBuilder struct {
	items []itemView
	data  [][]byte // each entry becomes one compressed data block
}

func newDatafileBuilder() *datafileBuilder {
	return &datafileBuilder{}
}

// addData appends a data block and returns its index.
func (b *datafileBuilder) addData(d []byte) int32 {
	idx := int32(len(b.data))
	cp := make([]byte, len(d))
	copy(cp, d)
	b.data = append(b.data, cp)
	return idx
}

// addStringData appends a NUL-terminated string as a data block.
func (b *datafileBuilder) addStringData(s string) int32 {
	buf := make([]byte, len(s)+1) // +1 for NUL
	copy(buf, s)
	return b.addData(buf)
}

// addItem appends an item.
func (b *datafileBuilder) addItem(typeID uint16, id uint16, data []int32) {
	b.items = append(b.items, itemView{
		TypeID: typeID,
		ID:     id,
		Data:   data,
	})
}

// finish builds the complete datafile and writes it to w.
func (b *datafileBuilder) finish(w io.Writer) error {
	df := b.buildDatafile()
	return df.writeTo(w)
}

func (b *datafileBuilder) buildDatafile() *datafile {
	// Compress data blocks and build offsets/sizes
	compressedBlocks := make([][]byte, len(b.data))
	dataOffsets := make([]int32, len(b.data))
	dataSizes := make([]int32, len(b.data))
	offset := int32(0)
	for i, raw := range b.data {
		compressed := zlibCompress(raw)
		compressedBlocks[i] = compressed
		dataOffsets[i] = offset
		dataSizes[i] = int32(len(raw))
		offset += int32(len(compressed))
	}

	// Build raw data section
	rawData := make([]byte, 0, offset)
	for _, block := range compressedBlocks {
		rawData = append(rawData, block...)
	}

	// Build item types (group items by type, sorted by type_id)
	typeMap := map[uint16][]int{} // typeID -> item indices
	for i, item := range b.items {
		typeMap[item.TypeID] = append(typeMap[item.TypeID], i)
	}
	typeIDs := make([]uint16, 0, len(typeMap))
	for tid := range typeMap {
		typeIDs = append(typeIDs, tid)
	}
	slices.Sort(typeIDs)

	// Reorder items by type
	orderedItems := make([]itemView, 0, len(b.items))
	itemTypes := make([]itemType, 0, len(typeIDs))
	for _, tid := range typeIDs {
		indices := typeMap[tid]
		it := itemType{
			TypeID: int32(tid),
			Start:  int32(len(orderedItems)),
			Num:    int32(len(indices)),
		}
		itemTypes = append(itemTypes, it)
		for _, idx := range indices {
			orderedItems = append(orderedItems, b.items[idx])
		}
	}

	return &datafile{
		version:     datafileVersion4,
		itemTypes:   itemTypes,
		items:       orderedItems,
		numData:     len(b.data),
		dataOffsets: dataOffsets,
		dataSizes:   dataSizes,
		rawData:     rawData,
	}
}

// ── info writing ─────────────────────────────────────────────────────────────

func (m *Map) writeInfo(b *datafileBuilder) {
	// item.Data[0]=version, [1]=author, [2]=mapversion, [3]=credits, [4]=license, [5]=settings
	authorIdx := b.addStringData(m.Info.Author)
	versionIdx := b.addStringData(m.Info.Version)
	creditsIdx := b.addStringData(m.Info.Credits)
	licenseIdx := b.addStringData(m.Info.License)

	data := []int32{1, authorIdx, versionIdx, creditsIdx, licenseIdx}

	if len(m.Info.Settings) > 0 {
		settingsIdx := b.addData(encodeCStringArray(m.Info.Settings))
		data = append(data, settingsIdx)
	}

	b.addItem(mapItemTypeInfo, 0, data)
}

func encodeCStringArray(ss []string) []byte {
	var buf []byte
	for _, s := range ss {
		buf = append(buf, []byte(s)...)
		buf = append(buf, 0)
	}
	return buf
}

// ── image writing ────────────────────────────────────────────────────────────

func (m *Map) writeImages(b *datafileBuilder) {
	for i, img := range m.Images {
		nameIdx := b.addStringData(img.Name)
		dataIdx := int32(-1)
		external := int32(0)
		if img.External {
			external = 1
		}
		if !img.External && img.RGBA != nil {
			dataIdx = b.addData(img.RGBA.Pix)
		}
		data := []int32{1, int32(img.Width), int32(img.Height), external, nameIdx, dataIdx}
		b.addItem(mapItemTypeImage, uint16(i), data)
	}
}

// ── envelope writing ─────────────────────────────────────────────────────────

func (m *Map) writeEnvelopes(b *datafileBuilder) {
	if len(m.Envelopes) == 0 {
		return
	}

	// Build global env points array
	var allPoints []int32
	pointOffset := 0
	for i, env := range m.Envelopes {
		startPoint := pointOffset
		for _, pt := range env.Points {
			allPoints = append(allPoints, pt.Time, int32(pt.CurveType),
				pt.Values[0], pt.Values[1], pt.Values[2], pt.Values[3])
		}
		pointOffset += len(env.Points)

		// Envelope item: [version, channels, startPoint, numPoints, name[8], synchronized]
		data := make([]int32, 13)
		data[0] = 2 // version (v2 = synchronized)
		data[1] = env.Channels
		data[2] = int32(startPoint)
		data[3] = int32(len(env.Points))
		nameI32 := encodeI32String(env.Name, 8)
		copy(data[4:12], nameI32)
		if env.Synchronized {
			data[12] = 1
		}
		b.addItem(mapItemTypeEnvelope, uint16(i), data)
	}

	// Single env_points item containing all points
	if len(allPoints) > 0 {
		b.addItem(mapItemTypeEnvPoints, 0, allPoints)
	}
}

// ── sound writing ────────────────────────────────────────────────────────────

func (m *Map) writeSounds(b *datafileBuilder) {
	for i, snd := range m.Sounds {
		nameIdx := b.addStringData(snd.Name)
		dataIdx := int32(-1)
		dataSize := int32(0)
		external := int32(0)
		if len(snd.Data) > 0 {
			dataIdx = b.addData(snd.Data)
			dataSize = int32(len(snd.Data))
		}
		data := []int32{1, external, nameIdx, dataIdx, dataSize}
		b.addItem(mapItemTypeSound, uint16(i), data)
	}
}

// ── group & layer writing ────────────────────────────────────────────────────

func (m *Map) writeGroupsAndLayers(b *datafileBuilder) {
	layerID := uint16(0)
	for gi, g := range m.Groups {
		startLayer := layerID
		for _, l := range g.Layers {
			writeLayer(b, &l, layerID)
			layerID++
		}

		// Group item: [version, offsetX, offsetY, parallaxX, parallaxY, startLayer, numLayers,
		//              useClipping, clipX, clipY, clipW, clipH, name[3]]
		gData := make([]int32, 15)
		gData[0] = 3 // version (v3 = name)
		gData[1] = g.OffsetX
		gData[2] = g.OffsetY
		gData[3] = g.ParallaxX
		gData[4] = g.ParallaxY
		gData[5] = int32(startLayer)
		gData[6] = int32(len(g.Layers))
		if g.Clipping {
			gData[7] = 1
		}
		gData[8] = g.ClipX
		gData[9] = g.ClipY
		gData[10] = g.ClipW
		gData[11] = g.ClipH
		nameI32 := encodeI32String(g.Name, 3)
		copy(gData[12:15], nameI32)

		b.addItem(mapItemTypeGroup, uint16(gi), gData)
	}
}

func writeLayer(b *datafileBuilder, l *Layer, id uint16) {
	switch l.Kind {
	case LayerKindQuads:
		writeQuadsLayer(b, l, id)
	case LayerKindSounds:
		writeSoundsLayer(b, l, id)
	default:
		writeTilemapLayer(b, l, id)
	}
}

func writeTilemapLayer(b *datafileBuilder, l *Layer, id uint16) {
	tileFlags := tileLayerFlags(l.Kind)

	// Encode base tile data
	tileDataIdx := int32(-1)
	if len(l.Tiles) > 0 {
		tileDataIdx = b.addData(encodeTiles(l.Tiles))
	}

	// layer item: [layerVersion, type, flags,
	//   tilemapVersion, width, height, tileFlags,
	//   colorR, colorG, colorB, colorA, colorEnv, colorEnvOffset,
	//   image, data, name[3],
	//   tele, speedup, front, switch, tune]
	data := make([]int32, 23)
	data[0] = 0 // layer version
	data[1] = layerTypeTilemap
	if l.Detail {
		data[2] = 1
	}
	data[3] = 3 // tilemap version (v3 = name + DDNet)
	data[4] = int32(l.Width)
	data[5] = int32(l.Height)
	data[6] = int32(tileFlags)
	data[7] = int32(l.ColorR)
	data[8] = int32(l.ColorG)
	data[9] = int32(l.ColorB)
	data[10] = int32(l.ColorA)
	data[11] = l.ColorEnv
	data[12] = l.ColorEnvOffset
	data[13] = int32(l.ImageID)
	data[14] = tileDataIdx

	nameI32 := encodeI32String(l.Name, 3)
	copy(data[15:18], nameI32)

	// DDNet special layer data indices at [18..22]
	data[18] = -1 // tele
	data[19] = -1 // speedup
	data[20] = -1 // front
	data[21] = -1 // switch
	data[22] = -1 // tune

	switch l.Kind {
	case LayerKindTele:
		if len(l.TeleTiles) > 0 {
			data[18] = b.addData(encodeTeleTiles(l.TeleTiles))
		}
	case LayerKindSpeedup:
		if len(l.SpeedupTiles) > 0 {
			data[19] = b.addData(encodeSpeedupTiles(l.SpeedupTiles))
		}
	case LayerKindFront:
		if len(l.Tiles) > 0 {
			data[20] = b.addData(encodeTiles(l.Tiles))
			// Clear base data index for front (front tiles are in the DDNet slot)
			data[14] = -1
		}
	case LayerKindSwitch:
		if len(l.SwitchTiles) > 0 {
			data[21] = b.addData(encodeSwitchTiles(l.SwitchTiles))
		}
	case LayerKindTune:
		if len(l.TuneTiles) > 0 {
			data[22] = b.addData(encodeTuneTiles(l.TuneTiles))
		}
	}

	b.addItem(mapItemTypeLayer, id, data)
}

func writeQuadsLayer(b *datafileBuilder, l *Layer, id uint16) {
	quadDataIdx := int32(-1)
	if len(l.Quads) > 0 {
		quadDataIdx = b.addData(encodeQuads(l.Quads))
	}

	// [layerVersion, type, flags, quadsVersion, numQuads, data, image, name[3]]
	data := make([]int32, 10)
	data[0] = 0 // layer version
	data[1] = layerTypeQuads
	if l.Detail {
		data[2] = 1
	}
	data[3] = 2 // quads version (v2 = name)
	data[4] = int32(len(l.Quads))
	data[5] = quadDataIdx
	data[6] = int32(l.QuadImageID)
	nameI32 := encodeI32String(l.Name, 3)
	copy(data[7:10], nameI32)

	b.addItem(mapItemTypeLayer, id, data)
}

func writeSoundsLayer(b *datafileBuilder, l *Layer, id uint16) {
	srcDataIdx := int32(-1)
	if len(l.SoundSources) > 0 {
		srcDataIdx = b.addData(encodeSoundSources(l.SoundSources))
	}

	// [layerVersion, type, flags, soundsVersion, numSources, data, sound, name[3]]
	data := make([]int32, 10)
	data[0] = 0 // layer version
	data[1] = layerTypeDdraceSounds
	if l.Detail {
		data[2] = 1
	}
	data[3] = 1 // sounds version
	data[4] = int32(len(l.SoundSources))
	data[5] = srcDataIdx
	data[6] = int32(l.SoundID)
	nameI32 := encodeI32String(l.Name, 3)
	copy(data[7:10], nameI32)

	b.addItem(mapItemTypeLayer, id, data)
}

// ── encoding helpers ─────────────────────────────────────────────────────────

func tileLayerFlags(kind LayerKind) uint32 {
	switch kind {
	case LayerKindGame:
		return tileLayerFlagGame
	case LayerKindFront:
		return tileLayerFlagFront
	case LayerKindTele:
		return tileLayerFlagTele
	case LayerKindSpeedup:
		return tileLayerFlagSpeedup
	case LayerKindSwitch:
		return tileLayerFlagSwitch
	case LayerKindTune:
		return tileLayerFlagTune
	default:
		return 0
	}
}

func encodeTiles(tiles []Tile) []byte {
	buf := make([]byte, len(tiles)*4)
	for i, t := range tiles {
		off := i * 4
		buf[off] = t.ID
		buf[off+1] = t.Flags
		buf[off+2] = 0 // skip
		buf[off+3] = 0 // unused
	}
	return buf
}

func encodeTeleTiles(tiles []TeleTile) []byte {
	buf := make([]byte, len(tiles)*2)
	for i, t := range tiles {
		off := i * 2
		buf[off] = t.Number
		buf[off+1] = t.ID
	}
	return buf
}

func encodeSpeedupTiles(tiles []SpeedupTile) []byte {
	buf := make([]byte, len(tiles)*6)
	for i, t := range tiles {
		off := i * 6
		buf[off] = t.Force
		buf[off+1] = t.MaxSpeed
		buf[off+2] = t.ID
		buf[off+3] = 0 // padding
		binary.LittleEndian.PutUint16(buf[off+4:off+6], uint16(t.Angle))
	}
	return buf
}

func encodeSwitchTiles(tiles []SwitchTile) []byte {
	buf := make([]byte, len(tiles)*4)
	for i, t := range tiles {
		off := i * 4
		buf[off] = t.Number
		buf[off+1] = t.ID
		buf[off+2] = t.Flags
		buf[off+3] = t.Delay
	}
	return buf
}

func encodeTuneTiles(tiles []TuneTile) []byte {
	buf := make([]byte, len(tiles)*2)
	for i, t := range tiles {
		off := i * 2
		buf[off] = t.Number
		buf[off+1] = t.ID
	}
	return buf
}

func encodeQuads(quads []Quad) []byte {
	buf := make([]byte, len(quads)*quadBinarySize)
	for qi, q := range quads {
		off := qi * quadBinarySize
		for i := range 5 {
			writePointBytes(buf[off:], q.Points[i])
			off += 8
		}
		for i := range 4 {
			writeColorBytes(buf[off:], q.Colors[i])
			off += 16
		}
		for i := range 4 {
			writeTexCoordBytes(buf[off:], q.TexCoords[i])
			off += 8
		}
		writeI32Bytes(buf[off:], q.PosEnv)
		writeI32Bytes(buf[off+4:], q.PosEnvOffset)
		writeI32Bytes(buf[off+8:], q.ColorEnv)
		writeI32Bytes(buf[off+12:], q.ColorEnvOffset)
	}
	return buf
}

func encodeSoundSources(sources []SoundSource) []byte {
	buf := make([]byte, len(sources)*soundSourceSize)
	for i, s := range sources {
		off := i * soundSourceSize
		writePointBytes(buf[off:], s.Position)
		loopV := int32(0)
		if s.Loop {
			loopV = 1
		}
		panV := int32(0)
		if s.Panning {
			panV = 1
		}
		writeI32Bytes(buf[off+8:], loopV)
		writeI32Bytes(buf[off+12:], panV)
		writeI32Bytes(buf[off+16:], s.Delay)
		writeI32Bytes(buf[off+20:], int32(s.Falloff))
		writeI32Bytes(buf[off+24:], s.PosEnv)
		writeI32Bytes(buf[off+28:], s.PosEnvOffset)
		writeI32Bytes(buf[off+32:], s.SoundEnv)
		writeI32Bytes(buf[off+36:], s.SoundEnvOffset)
		writeI32Bytes(buf[off+40:], s.ShapeType)
		writeI32Bytes(buf[off+44:], s.ShapeWidth)
		writeI32Bytes(buf[off+48:], s.ShapeHeight)
	}
	return buf
}

func writePointBytes(buf []byte, p Point) {
	x := int32(p.X * 32768.0)
	y := int32(p.Y * 32768.0)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(x))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(y))
}

func writeTexCoordBytes(buf []byte, p Point) {
	x := int32(p.X * 1024.0)
	y := int32(p.Y * 1024.0)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(x))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(y))
}

func writeColorBytes(buf []byte, c color.NRGBA) {
	binary.LittleEndian.PutUint32(buf[0:4], uint32(c.R))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(c.G))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(c.B))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(c.A))
}

func writeI32Bytes(buf []byte, v int32) {
	binary.LittleEndian.PutUint32(buf[0:4], uint32(v))
}

// encodeI32String encodes a string into the Teeworlds "n x int32" format
// (big-endian, XOR -128). numInt32s specifies the number of int32 slots.
func encodeI32String(s string, numInt32s int) []int32 {
	buf := make([]byte, numInt32s*4)
	if s == "" {
		result := make([]int32, numInt32s)
		return result
	}
	// XOR encode
	for i := 0; i < len(s) && i < len(buf)-1; i++ {
		buf[i] = byte(int(s[i]) - 128)
	}
	// The last byte should be NUL (0 after XOR = byte(-128) => 128)
	// Actually: decoding does buf[i] = byte(int(buf[i]) + 128)
	// So to encode char c: encoded = byte(c - 128) = byte(int(c) - 128)
	// NUL terminator: 0 - 128 = -128 = 128 unsigned. But the buf is already zeroed.
	// The decoder strips trailing NULs after XOR. The trailing zero bytes
	// XOR-decode to +128 = 0x80 which is NOT zero. Actually, the decoder adds 128:
	// byte(int(0) + 128) = 128, which is not NUL. So zero bytes in the encoded form
	// become 128 after decode, and the decoder strips trailing NULs (value 0 after decode).
	// For the NUL terminator, we need the encoded value to decode to 0:
	// encode(0) = byte(0 - 128) = 128. So we set buf[len(s)] = 128.
	if len(s) < len(buf) {
		buf[len(s)] = 128
	}

	result := make([]int32, numInt32s)
	for i := range numInt32s {
		result[i] = int32(binary.BigEndian.Uint32(buf[i*4 : i*4+4]))
	}
	return result
}

// zlibCompress compresses data using zlib (default compression).
func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}
