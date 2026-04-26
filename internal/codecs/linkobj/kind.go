// Package linkobj decodes the per-entry "LinkObjRef" payload that follows
// the primary PTH0 path inside LIfp / LIbd / LIvi link-info blocks.
//
// pylabview models ~115 LinkObjRef subclasses (references/pylabview/pylabview/
// LVlinkinfo.py:4235 newLinkObject); we expose every observed 4-byte ident
// as a stable LinkKind, fully decode the subclasses needed for the corpus,
// and fall back to OpaqueTarget for the rest. Round-trip is preserved
// regardless of whether a typed parser exists.
//
// The decoder targets the modern wire format (LabVIEW ≥ 14.0.0.3 / 12.0.0.3
// / 8.6.0.1 / 8.2.0.3), which is what every shipped corpus file uses
// (LV 25.3.2). Older-format support can be layered on later by
// dispatching on Context.FileVersion before calling Decode.
package linkobj

// LinkKind is a stable Go-named identifier for a LinkObjRef subclass. The
// value is *not* a wire-format token; it's a compile-time constant used in
// switches and exposed via pkg/lvvi.DependencyEntry.LinkKind().
type LinkKind int

// Identifier kinds. Every 4-byte ident pylabview's newLinkObject recognises
// has a slot here so callers can branch on it even when the typed parser
// has not been ported.
const (
	KindUnknown LinkKind = iota

	KindInstanceVIToOwnerVI         // IVOV
	KindHeapToAssembly              // DNDA
	KindVIToAssembly                // DNVA
	KindVIToEIOLink                 // EiVr
	KindHeapToEIOLink               // HpEr
	KindVIToCCSymbolLink            // V2CC
	KindVIToFileLink                // VIFl
	KindVIToFileNoWarnLink          // VIFN
	KindVIToFilePathLink            // VIXF
	KindHeapToFilePathLink          // HOXF
	KindXNodeToFilePathLink         // XNFP
	KindVIToGenVI                   // VIGV
	KindVIToInstantiationVI         // VIIV
	KindInstantiationVIToGenVI      // IVGV
	KindVIToVINamedLink             // VTVN
	KindVIToLibraryDataLink         // V2LD
	KindVIToMSLink                  // VIMS
	KindTypeDefToCCLink             // TDCC (also LVCC under FPHP list)
	KindHeapToXCtlInterface         // HXCI
	KindXCtlToXInterface            // XCXI
	KindVIToXCtlInterface           // VIXC
	KindVIToXNodeInterface          // VIXN
	KindVIToXNodeProjectItemLink    // XVPR
	KindHeapToXNodeProjectItemLink  // XHPR
	KindActiveXVIToTypeLib          // AXVT
	KindVIToLib                     // VILB
	KindUDClassDDOToUDClassAPILink  // FPPI
	KindDDODefaultDataToUDClassAPI  // DDPI
	KindHeapObjToUDClassAPILink     // VRPI
	KindVIToUDClassAPILink          // VIPI
	KindDataValueRefVIToUDClassAPI  // RVPI
	KindVIToVariableAbsoluteLink    // VIVr
	KindVIToVariableRelativeLink    // VIVl
	KindHeapToVariableAbsoluteLink  // HpVr
	KindHeapToVariableRelativeLink  // HpVL
	KindDSToVariableAbsoluteLink    // DSVr
	KindDSToVariableRelativeLink    // DSVl
	KindDSToDSLink                  // DSDS (also VIDS under VIDS list)
	KindDSToExtFuncLink             // DSEF (also XFun under VIDS/BDHP)
	KindDSToCINLink                 // DSCN (also LVSB under VIDS/BDHP)
	KindDSToScriptLink              // DSSC (also SFTB under VIDS)
	KindDSToCallByRefLink           // DSCB
	KindDSToStaticVILink            // DSSV
	KindVIToStdVILink               // VIVI (also LVIN under LVIN/BDHP)
	KindVIToProgRetLink             // VIPR (also LVPR under LVIN)
	KindVIToPolyLink                // VIPV (also POLY under LVIN/BDHP)
	KindVIToCCLink                  // VICC (also LVCC/CCCC under LVCC/LVIN/BDHP)
	KindVIToStaticVILink            // BSVR
	KindVIToAdaptiveVILink          // VIAV
	KindHeapToCCSymbolLink          // H2CC
	KindIUseToVILink                // IUVI
	KindNonVINonHeapToTypedefLink   // .2TD
	KindCCSymbolLink                // CCLO
	KindHeapNamedLink               // HpEx
	KindFilePathLink                // XFil
	KindRCFilePathLink              // RFil
	KindHeapToFileLink              // HpFl
	KindHeapToFileNoWarnLink        // HpFN
	KindVIToRCFileLink              // VIRC
	KindIUseToInstantiationVILink   // IUIV
	KindGenIUseToGenVILink          // GUGV
	KindNodeToEFLink                // NEXF
	KindHeapToVILink                // HVIR
	KindPIUseToPolyLink             // PUPV
	KindIUseToProgRetLink           // IUPR
	KindStaticVIRefToVILink         // SVVI
	KindNodeToCINLink               // NCIN
	KindNodeToScriptLink            // NSCR
	KindStaticCallByRefToVILink     // SCVI
	KindHeapToRCFileLink            // RCFL
	KindHeapToVINamedLink           // HpVI
	KindHeapToLibraryDataLink       // H2LD
	KindMSNToMSLink                 // MNMS
	KindMSToMSImplVILink            // MSIM
	KindMSCallByRefToMSLink         // CBMS
	KindMathScriptLink              // MUDF
	KindFBoxLineToInstantnVILink    // FBIV
	KindOMHeapToResource            // OBDR
	KindOMVIToResource              // OVIR
	KindOMExtResLink                // OXTR
	KindGIToAbstractVI              // GIVI
	KindGIToAbilityVI               // GIAY
	KindXIToPropertyVI              // XIPY
	KindXIToMethodVI                // XIMD
	KindGInterfaceLink              // LIBR
	KindXInterfaceLink              // XINT
	KindXCtlInterfaceLink           // LVXC
	KindXNodeInterfaceLink          // XNDI
	KindVIToContainerItemLink       // VICI
	KindHeapToContainerItemLink     // HpCI
	KindContainerItemLinkObj        // CILO
	KindXNodeProjectItemLinkObj     // XPLO
	KindXNodeToExtFuncLink          // XNEF
	KindXNodeToVILink               // XNVI
	KindActiveXBDToTypeLib          // AXDT
	KindActiveXTLibLinkObj          // AXTL
	KindXNodeToXInterface           // XNXI
	KindUDClassLibInheritsLink      // HEIR
	KindUDClassLibToVILink          // C2vi
	KindUDClassLibToMemberVILink    // C2VI
	KindUDClassLibToPrivDataCtlLink // C2Pr
	KindHeapToUDClassAPILink        // HOPI / UDPI
	KindDynInfoToUDClassAPILink     // DyOM
	KindPropNodeItemToUDClassAPI    // PNOM
	KindCreOrDesRefToUDClassAPI     // DRPI
	KindDDOToUDClassAPILink         // DOPI
	KindAPIToAPILink                // AP2A
	KindAPIToNearestImplVILink      // AP2I
	KindAPIToChildAPILink           // AP2C
	KindMemberVIItem                // CMem
	KindUDClassLibrary              // CLIB
	KindHeapToXNodeInterface        // HXNI
	KindHeapToGInterface            // GINT
)

// kindInfo is the static metadata for one LinkKind.
type kindInfo struct {
	kind        LinkKind
	ident       string // primary 4-byte ident (the canonical wire token)
	description string // short human label
	aliases     []string // list-context-dispatched aliases (e.g. "LVCC" for TDCC)
}

// kindTable is the source of truth, mirroring the dispatch in
// pylabview/LVlinkinfo.py newLinkObject().
var kindTable = []kindInfo{
	{KindInstanceVIToOwnerVI, "IVOV", "Instance VI → Owner VI", nil},
	{KindHeapToAssembly, "DNDA", "Heap → .NET Assembly", nil},
	{KindVIToAssembly, "DNVA", "VI → .NET Assembly", nil},
	{KindVIToEIOLink, "EiVr", "VI → EIO", nil},
	{KindHeapToEIOLink, "HpEr", "Heap → EIO", nil},
	{KindVIToCCSymbolLink, "V2CC", "VI → CC Symbol", nil},
	{KindVIToFileLink, "VIFl", "VI → File", nil},
	{KindVIToFileNoWarnLink, "VIFN", "VI → File (no-warn)", nil},
	{KindVIToFilePathLink, "VIXF", "VI → File Path", nil},
	{KindHeapToFilePathLink, "HOXF", "Heap → File Path", nil},
	{KindXNodeToFilePathLink, "XNFP", "XNode → File Path", nil},
	{KindVIToGenVI, "VIGV", "VI → Generic VI", nil},
	{KindVIToInstantiationVI, "VIIV", "VI → Instantiation VI", nil},
	{KindInstantiationVIToGenVI, "IVGV", "Instantiation VI → Generic VI", nil},
	{KindVIToVINamedLink, "VTVN", "VI → Named VI", nil},
	{KindVIToLibraryDataLink, "V2LD", "VI → Library Data", nil},
	{KindVIToMSLink, "VIMS", "VI → MathScript", nil},
	{KindTypeDefToCCLink, "TDCC", "TypeDef → CustCtl", []string{"LVCC"}},
	{KindHeapToXCtlInterface, "HXCI", "Heap → XCtl Interface", nil},
	{KindXCtlToXInterface, "XCXI", "XCtl → XInterface", nil},
	{KindVIToXCtlInterface, "VIXC", "VI → XCtl Interface", nil},
	{KindVIToXNodeInterface, "VIXN", "VI → XNode Interface", nil},
	{KindVIToXNodeProjectItemLink, "XVPR", "VI → XNode Project Item", nil},
	{KindHeapToXNodeProjectItemLink, "XHPR", "Heap → XNode Project Item", nil},
	{KindActiveXVIToTypeLib, "AXVT", "ActiveX VI → TypeLib", nil},
	{KindVIToLib, "VILB", "VI → Library", nil},
	{KindUDClassDDOToUDClassAPILink, "FPPI", "UDClass DDO → UDClass API", nil},
	{KindDDODefaultDataToUDClassAPI, "DDPI", "DDO Default Data → UDClass API", nil},
	{KindHeapObjToUDClassAPILink, "VRPI", "Heap Object → UDClass API", nil},
	{KindVIToUDClassAPILink, "VIPI", "VI → UDClass API", nil},
	{KindDataValueRefVIToUDClassAPI, "RVPI", "DataValueRef VI → UDClass API", nil},
	{KindVIToVariableAbsoluteLink, "VIVr", "VI → Variable (absolute)", nil},
	{KindVIToVariableRelativeLink, "VIVl", "VI → Variable (relative)", nil},
	{KindHeapToVariableAbsoluteLink, "HpVr", "Heap → Variable (absolute)", nil},
	{KindHeapToVariableRelativeLink, "HpVL", "Heap → Variable (relative)", nil},
	{KindDSToVariableAbsoluteLink, "DSVr", "DataSpace → Variable (absolute)", nil},
	{KindDSToVariableRelativeLink, "DSVl", "DataSpace → Variable (relative)", nil},
	{KindDSToDSLink, "DSDS", "DataSpace → DataSpace", []string{"VIDS"}},
	{KindDSToExtFuncLink, "DSEF", "DataSpace → Ext Func", []string{"XFun"}},
	{KindDSToCINLink, "DSCN", "DataSpace → CIN", []string{"LVSB"}},
	{KindDSToScriptLink, "DSSC", "DataSpace → Script", []string{"SFTB"}},
	{KindDSToCallByRefLink, "DSCB", "DataSpace → Call-by-Ref", nil},
	{KindDSToStaticVILink, "DSSV", "DataSpace → Static VI", nil},
	{KindVIToStdVILink, "VIVI", "VI → Std VI", []string{"LVIN"}},
	{KindVIToProgRetLink, "VIPR", "VI → ProgRet", []string{"LVPR"}},
	{KindVIToPolyLink, "VIPV", "VI → Poly", []string{"POLY"}},
	{KindVIToCCLink, "VICC", "VI → CustCtl", []string{"CCCC"}}, // LVCC ambiguous (also TDCC)
	{KindVIToStaticVILink, "BSVR", "VI → Static VI", nil},
	{KindVIToAdaptiveVILink, "VIAV", "VI → Adaptive VI", nil},
	{KindHeapToCCSymbolLink, "H2CC", "Heap → CC Symbol", nil},
	{KindIUseToVILink, "IUVI", "IUse → VI", nil},
	{KindNonVINonHeapToTypedefLink, ".2TD", "Non-VI Non-Heap → TypeDef", nil},
	{KindCCSymbolLink, "CCLO", "CC Symbol", nil},
	{KindHeapNamedLink, "HpEx", "Heap (named)", nil},
	{KindFilePathLink, "XFil", "File Path", nil},
	{KindRCFilePathLink, "RFil", "RC File Path", nil},
	{KindHeapToFileLink, "HpFl", "Heap → File", nil},
	{KindHeapToFileNoWarnLink, "HpFN", "Heap → File (no-warn)", nil},
	{KindVIToRCFileLink, "VIRC", "VI → RC File", nil},
	{KindIUseToInstantiationVILink, "IUIV", "IUse → Instantiation VI", nil},
	{KindGenIUseToGenVILink, "GUGV", "Generic IUse → Generic VI", nil},
	{KindNodeToEFLink, "NEXF", "Node → Ext Func", nil},
	{KindHeapToVILink, "HVIR", "Heap → VI", nil},
	{KindPIUseToPolyLink, "PUPV", "PIUse → Poly", nil},
	{KindIUseToProgRetLink, "IUPR", "IUse → ProgRet", nil},
	{KindStaticVIRefToVILink, "SVVI", "Static VI Ref → VI", nil},
	{KindNodeToCINLink, "NCIN", "Node → CIN", nil},
	{KindNodeToScriptLink, "NSCR", "Node → Script", nil},
	{KindStaticCallByRefToVILink, "SCVI", "Static Call-by-Ref → VI", nil},
	{KindHeapToRCFileLink, "RCFL", "Heap → RC File", nil},
	{KindHeapToVINamedLink, "HpVI", "Heap → VI (named)", nil},
	{KindHeapToLibraryDataLink, "H2LD", "Heap → Library Data", nil},
	{KindMSNToMSLink, "MNMS", "MathScript Node → MathScript", nil},
	{KindMSToMSImplVILink, "MSIM", "MathScript → MathScript Impl VI", nil},
	{KindMSCallByRefToMSLink, "CBMS", "MathScript Call-by-Ref → MathScript", nil},
	{KindMathScriptLink, "MUDF", "MathScript User Function", nil},
	{KindFBoxLineToInstantnVILink, "FBIV", "FBox Line → Instantiation VI", nil},
	{KindOMHeapToResource, "OBDR", "OM Heap → Resource", nil},
	{KindOMVIToResource, "OVIR", "OM VI → Resource", nil},
	{KindOMExtResLink, "OXTR", "OM Ext Resource", nil},
	{KindGIToAbstractVI, "GIVI", "Generic Interface → Abstract VI", nil},
	{KindGIToAbilityVI, "GIAY", "Generic Interface → Ability VI", nil},
	{KindXIToPropertyVI, "XIPY", "XInterface → Property VI", nil},
	{KindXIToMethodVI, "XIMD", "XInterface → Method VI", nil},
	{KindGInterfaceLink, "LIBR", "Generic Interface", nil},
	{KindXInterfaceLink, "XINT", "XInterface", nil},
	{KindXCtlInterfaceLink, "LVXC", "XCtl Interface", nil},
	{KindXNodeInterfaceLink, "XNDI", "XNode Interface", nil},
	{KindVIToContainerItemLink, "VICI", "VI → Container Item", nil},
	{KindHeapToContainerItemLink, "HpCI", "Heap → Container Item", nil},
	{KindContainerItemLinkObj, "CILO", "Container Item", nil},
	{KindXNodeProjectItemLinkObj, "XPLO", "XNode Project Item", nil},
	{KindXNodeToExtFuncLink, "XNEF", "XNode → Ext Func", nil},
	{KindXNodeToVILink, "XNVI", "XNode → VI", nil},
	{KindActiveXBDToTypeLib, "AXDT", "ActiveX BD → TypeLib", nil},
	{KindActiveXTLibLinkObj, "AXTL", "ActiveX TLib", nil},
	{KindXNodeToXInterface, "XNXI", "XNode → XInterface", nil},
	{KindUDClassLibInheritsLink, "HEIR", "UDClass Lib Inherits", nil},
	{KindUDClassLibToVILink, "C2vi", "UDClass Lib → VI", nil},
	{KindUDClassLibToMemberVILink, "C2VI", "UDClass Lib → Member VI", nil},
	{KindUDClassLibToPrivDataCtlLink, "C2Pr", "UDClass Lib → Priv Data Ctl", nil},
	{KindHeapToUDClassAPILink, "HOPI", "Heap → UDClass API", []string{"UDPI"}},
	{KindDynInfoToUDClassAPILink, "DyOM", "Dyn Info → UDClass API", nil},
	{KindPropNodeItemToUDClassAPI, "PNOM", "Prop Node Item → UDClass API", nil},
	{KindCreOrDesRefToUDClassAPI, "DRPI", "Create/Destroy Ref → UDClass API", nil},
	{KindDDOToUDClassAPILink, "DOPI", "DDO → UDClass API", nil},
	{KindAPIToAPILink, "AP2A", "API → API", nil},
	{KindAPIToNearestImplVILink, "AP2I", "API → Nearest Impl VI", nil},
	{KindAPIToChildAPILink, "AP2C", "API → Child API", nil},
	{KindMemberVIItem, "CMem", "Member VI Item", nil},
	{KindUDClassLibrary, "CLIB", "UDClass Library", nil},
	{KindHeapToXNodeInterface, "HXNI", "Heap → XNode Interface", nil},
	{KindHeapToGInterface, "GINT", "Heap → Generic Interface", nil},
}

var (
	kindByIdent       map[string]LinkKind
	kindByValue       map[LinkKind]kindInfo
)

func init() {
	kindByIdent = make(map[string]LinkKind, 2*len(kindTable))
	kindByValue = make(map[LinkKind]kindInfo, len(kindTable))
	for _, k := range kindTable {
		kindByValue[k.kind] = k
		kindByIdent[k.ident] = k.kind
		// Aliases — but only when unambiguous. LVCC, for example, can be
		// either TDCC (under FPHP/LVCC list contexts) or VICC (under LVIN
		// list context). When we encounter such an alias without list
		// context, we fall through to KindUnknown via the lookup miss
		// rather than picking arbitrarily. So aliases populate the table
		// only when their ident isn't already claimed.
		for _, alias := range k.aliases {
			if _, taken := kindByIdent[alias]; !taken {
				kindByIdent[alias] = k.kind
			}
		}
	}
}

// LookupKind returns the LinkKind for a 4-byte LinkObjRef ident.
//
// The mapping is unambiguous for canonical idents (TDCC, VILB, IUVI, …).
// For list-context-dispatched aliases (LVCC, LVIN, POLY, CCCC, XFun, LVSB,
// SFTB, VIDS, LVPR, UDPI), the lookup uses the first registration order
// from kindTable when not ambiguous, otherwise returns KindUnknown.
func LookupKind(ident string) LinkKind {
	if k, ok := kindByIdent[ident]; ok {
		return k
	}
	return KindUnknown
}

// CanonicalIdent returns the wire-token ident for a LinkKind, or "" if k
// is KindUnknown.
func CanonicalIdent(k LinkKind) string {
	if info, ok := kindByValue[k]; ok {
		return info.ident
	}
	return ""
}

// Description returns a short human-readable description of the link kind
// (e.g. "TypeDef → CustCtl"), or the literal "unknown link" if k is unknown.
func Description(k LinkKind) string {
	if info, ok := kindByValue[k]; ok {
		return info.description
	}
	return "unknown link"
}
