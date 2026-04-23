// Package lvmeta provides Tier 2 metadata editing helpers layered on top of
// pkg/lvrsrc's preserving container model.
//
// Tier 2 edits are targeted mutations on understood resources or container
// metadata. They preserve untouched blocks, sections, names, and opaque bytes,
// and they are intended to remain distinct from Tier 1 preserving rewrites and
// future Tier 3 raw patching workflows.
package lvmeta
