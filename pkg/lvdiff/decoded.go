package lvdiff

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/CWBudde/lvrsrc/internal/codecs"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdex"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdpw"
	"github.com/CWBudde/lvrsrc/internal/codecs/bdse"
	"github.com/CWBudde/lvrsrc/internal/codecs/conpane"
	"github.com/CWBudde/lvrsrc/internal/codecs/dthp"
	"github.com/CWBudde/lvrsrc/internal/codecs/fpex"
	"github.com/CWBudde/lvrsrc/internal/codecs/fpse"
	"github.com/CWBudde/lvrsrc/internal/codecs/ftab"
	"github.com/CWBudde/lvrsrc/internal/codecs/hist"
	"github.com/CWBudde/lvrsrc/internal/codecs/icon"
	"github.com/CWBudde/lvrsrc/internal/codecs/libd"
	"github.com/CWBudde/lvrsrc/internal/codecs/libn"
	"github.com/CWBudde/lvrsrc/internal/codecs/lifp"
	"github.com/CWBudde/lvrsrc/internal/codecs/lvsr"
	"github.com/CWBudde/lvrsrc/internal/codecs/muid"
	"github.com/CWBudde/lvrsrc/internal/codecs/rtsg"
	"github.com/CWBudde/lvrsrc/internal/codecs/strg"
	"github.com/CWBudde/lvrsrc/internal/codecs/vctp"
	"github.com/CWBudde/lvrsrc/internal/codecs/vers"
	"github.com/CWBudde/lvrsrc/internal/codecs/vits"
	"github.com/CWBudde/lvrsrc/internal/codecs/vpdp"
	"github.com/CWBudde/lvrsrc/pkg/lvrsrc"
)

// contextFromFile derives the per-file codec Context used when invoking
// typed codecs during a diff. It mirrors pkg/lvmeta.contextFromFile so the
// two dispatch paths agree on how FormatVersion and Kind reach codecs.
func contextFromFile(f *lvrsrc.File) codecs.Context {
	if f == nil {
		return codecs.Context{}
	}
	return codecs.Context{
		FileVersion: f.Header.FormatVersion,
		Kind:        f.Kind,
	}
}

type blobSummary struct {
	Size int    `json:"size"`
	Hash string `json:"hash"`
}

func defaultDecodedDiffers(a, b *lvrsrc.File) map[string]DecodedDiffer {
	r := codecs.New()
	r.Register(conpane.PointerCodec{})
	r.Register(conpane.CountCodec{})
	r.Register(icon.MonoCodec{})
	r.Register(icon.Color4Codec{})
	r.Register(icon.Color8Codec{})
	r.Register(libd.Codec{})
	r.Register(lifp.Codec{})
	r.Register(strg.Codec{})
	r.Register(vers.Codec{})
	r.Register(vctp.Codec{})
	r.Register(lvsr.Codec{})
	r.Register(muid.Codec{})
	r.Register(fpse.Codec{})
	r.Register(bdse.Codec{})
	r.Register(vpdp.Codec{})
	r.Register(dthp.Codec{})
	r.Register(rtsg.Codec{})
	r.Register(libn.Codec{})
	r.Register(hist.Codec{})
	r.Register(bdpw.Codec{})
	r.Register(fpex.Codec{})
	r.Register(bdex.Codec{})
	r.Register(ftab.Codec{})
	r.Register(vits.Codec{})

	aCtx := contextFromFile(a)
	bCtx := contextFromFile(b)

	out := make(map[string]DecodedDiffer)
	for _, cap := range r.Capabilities() {
		codec := r.Lookup(cap.FourCC)
		out[cap.FourCC] = makeCodecDiffer(codec, aCtx, bCtx)
	}
	return out
}

func makeCodecDiffer(codec codecs.ResourceCodec, aCtx, bCtx codecs.Context) DecodedDiffer {
	return func(blockType string, sectionIndex int32, oldPayload, newPayload []byte) []DiffItem {
		prefix := fmt.Sprintf("blocks.%s/%d.decoded", blockType, sectionIndex)
		oldValue, oldErr := codec.Decode(aCtx, oldPayload)
		newValue, newErr := codec.Decode(bCtx, newPayload)
		switch {
		case oldErr == nil && newErr == nil:
			return diffDecodedValues(prefix, reflect.ValueOf(oldValue), reflect.ValueOf(newValue))
		case oldErr != nil && newErr != nil && oldErr.Error() == newErr.Error():
			return nil
		default:
			return []DiffItem{{
				Kind:     KindDecoded,
				Category: CategoryModified,
				Path:     prefix,
				Old:      decodeResult(oldValue, oldErr),
				New:      decodeResult(newValue, newErr),
				Message:  fmt.Sprintf("%s decoded value changed", prefix),
			}}
		}
	}
}

func decodeResult(v any, err error) any {
	if err != nil {
		return err.Error()
	}
	return snapshotValue(reflect.ValueOf(v))
}

func diffDecodedValues(path string, oldV, newV reflect.Value) []DiffItem {
	oldV = unwrapValue(oldV)
	newV = unwrapValue(newV)

	switch {
	case !oldV.IsValid() && !newV.IsValid():
		return nil
	case !oldV.IsValid():
		return []DiffItem{{
			Kind:     KindDecoded,
			Category: CategoryAdded,
			Path:     path,
			New:      snapshotValue(newV),
			Message:  fmt.Sprintf("%s decoded value added", path),
		}}
	case !newV.IsValid():
		return []DiffItem{{
			Kind:     KindDecoded,
			Category: CategoryRemoved,
			Path:     path,
			Old:      snapshotValue(oldV),
			Message:  fmt.Sprintf("%s decoded value removed", path),
		}}
	}

	if oldV.Type() != newV.Type() {
		return []DiffItem{{
			Kind:     KindDecoded,
			Category: CategoryModified,
			Path:     path,
			Old:      snapshotValue(oldV),
			New:      snapshotValue(newV),
			Message:  fmt.Sprintf("%s decoded type changed", path),
		}}
	}

	if oldV.Type() == reflect.TypeOf([]byte(nil)) {
		oldBytes := oldV.Bytes()
		newBytes := newV.Bytes()
		if bytes.Equal(oldBytes, newBytes) {
			return nil
		}
		return []DiffItem{{
			Kind:     KindDecoded,
			Category: CategoryModified,
			Path:     path,
			Old:      summarizeBlob(oldBytes),
			New:      summarizeBlob(newBytes),
			Message:  fmt.Sprintf("%s decoded bytes changed", path),
		}}
	}

	switch oldV.Kind() {
	case reflect.Struct:
		var items []DiffItem
		t := oldV.Type()
		for i := 0; i < oldV.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" || ignoreDecodedField(t, field.Name) {
				continue
			}
			items = append(items, diffDecodedValues(path+"."+field.Name, oldV.Field(i), newV.Field(i))...)
		}
		return items
	case reflect.Slice:
		oldLen, newLen := oldV.Len(), newV.Len()
		common := oldLen
		if newLen < common {
			common = newLen
		}
		var items []DiffItem
		for i := 0; i < common; i++ {
			items = append(items, diffDecodedValues(fmt.Sprintf("%s[%d]", path, i), oldV.Index(i), newV.Index(i))...)
		}
		for i := common; i < oldLen; i++ {
			items = append(items, DiffItem{
				Kind:     KindDecoded,
				Category: CategoryRemoved,
				Path:     fmt.Sprintf("%s[%d]", path, i),
				Old:      snapshotValue(oldV.Index(i)),
				Message:  fmt.Sprintf("%s[%d] decoded item removed", path, i),
			})
		}
		for i := common; i < newLen; i++ {
			items = append(items, DiffItem{
				Kind:     KindDecoded,
				Category: CategoryAdded,
				Path:     fmt.Sprintf("%s[%d]", path, i),
				New:      snapshotValue(newV.Index(i)),
				Message:  fmt.Sprintf("%s[%d] decoded item added", path, i),
			})
		}
		return items
	default:
		if reflect.DeepEqual(oldV.Interface(), newV.Interface()) {
			return nil
		}
		return []DiffItem{{
			Kind:     KindDecoded,
			Category: CategoryModified,
			Path:     path,
			Old:      snapshotValue(oldV),
			New:      snapshotValue(newV),
			Message:  fmt.Sprintf("%s decoded value changed", path),
		}}
	}
}

func unwrapValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func snapshotValue(v reflect.Value) any {
	v = unwrapValue(v)
	if !v.IsValid() {
		return nil
	}
	if v.Type() == reflect.TypeOf([]byte(nil)) {
		return summarizeBlob(v.Bytes())
	}
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		out := make(map[string]any)
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" || ignoreDecodedField(t, field.Name) {
				continue
			}
			out[field.Name] = snapshotValue(v.Field(i))
		}
		return out
	case reflect.Slice:
		out := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = snapshotValue(v.Index(i))
		}
		return out
	default:
		return v.Interface()
	}
}

func summarizeBlob(b []byte) blobSummary {
	sum := sha256.Sum256(b)
	return blobSummary{
		Size: len(b),
		Hash: hex.EncodeToString(sum[:]),
	}
}

func ignoreDecodedField(t reflect.Type, name string) bool {
	return t.PkgPath() == "github.com/CWBudde/lvrsrc/internal/codecs/vctp" &&
		t.Name() == "Value" &&
		name == "Compressed"
}
