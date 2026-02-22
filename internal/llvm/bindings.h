#include "llvm-c/Core.h"
#include "llvm-c/TargetMachine.h"

#ifdef __cplusplus
#include "llvm/ADT/StringRef.h"
extern "C" {
#endif

// LLVMGoStringRef is used to pass strings between Go and LLVM.
// The _GoString_ type is only available in the cgo prologue (and is also partially broken until 1.25), so we cannot use it.
// This type can be trivially cast between go strings and llvm::StringRef.
typedef struct {
	const char* ptr;
	size_t len;
} LLVMGoStringRef;

// goCloneString copies a string to the Go heap.
// dst is a pointer to a Go string variable.
void goCloneString(void* dst, LLVMGoStringRef src);

// Attribute sets cannot be passed directly.
// The context instead owns opaque pointers.
typedef struct LLVMGoOpaqueAttributeSet *LLVMGoAttributeSetRef;
typedef struct LLVMGoOpaqueAttributeList *LLVMGoAttributeListRef;

// Contexts
LLVMContextRef LLVMGoContextCreate();
void LLVMGoContextDestroy(LLVMContextRef ctx);

typedef enum {
	LLVMGoRelocModelDefault,
	LLVMGoRelocModelStatic,
	LLVMGoRelocModelPIC,
	LLVMGoRelocModelDynamicNoPIC,
	LLVMGoRelocModelROPI,
	LLVMGoRelocModelRWPI,
	LLVMGoRelocModelROPIRWPI,
} LLVMGoRelocationModel;
typedef enum {
	LLVMGoCodeModelDefault,
	LLVMGoCodeModelTiny,
	LLVMGoCodeModelSmall,
	LLVMGoCodeModelKernel,
	LLVMGoCodeModelMedium,
	LLVMGoCodeModelLarge,
} LLVMGoCodeModel;
typedef struct {
	LLVMGoStringRef triple;
	LLVMGoStringRef cpu;
	LLVMGoStringRef features;
	LLVMGoRelocationModel relocationModel;
	LLVMGoCodeModel codeModel;
	int codeGenLevel;
} LLVMGoTargetMachineConfig;
LLVMTargetMachineRef LLVMGoCreateTargetMachine(
	LLVMGoTargetMachineConfig config,
	void* errMsgDst
);

typedef struct {
	unsigned program;
	unsigned global;
	unsigned alloca;
} LLVMGoAddressSpaces;

LLVMGoAddressSpaces LLVMGoGetAddressSpaces(LLVMTargetDataRef dataLayout);

typedef struct {
	uint64_t size;
	uint64_t align;
	bool scalable;
} LLVMGoABIType;

LLVMGoABIType LLVMGoGetABIType(LLVMTargetDataRef dataLayout, LLVMTypeRef type);

LLVMModuleRef LLVMGoNewModule(
	LLVMContextRef ctx,
	LLVMGoStringRef name,
	LLVMGoStringRef triple,
	LLVMTargetDataRef dataLayout
);
// LLVM's C bindings have an equivalent API, but not until LLVM 20.
LLVMValueRef LLVMGoGetNamedValue(LLVMModuleRef mod, LLVMGoStringRef str);
typedef enum {
	LLVMGoLinkageExternal,
	LLVMGoLinkageAvailableExternally,
	LLVMGoLinkageOnceAny,
	LLVMGoLinkageOnceODR,
	LLVMGoLinkageWeakAny,
	LLVMGoLinkageWeakODR,
	LLVMGoLinkageAppending,
	LLVMGoLinkageInternal,
	LLVMGoLinkagePrivate,
	LLVMGoLinkageExternalWeak,
	LLVMGoLinkageCommon,
} LLVMGoLinkage;
typedef enum {
	LLVMGoUnnamedAddrUnique,
	LLVMGoUnnamedAddrLocal,
	LLVMGoUnnamedAddrGlobal,
} LLVMGoUnnamedAddr;
typedef enum {
	LLVMGoVisibilityDefault,
	LLVMGoVisibilityHidden,
	LLVMGoVisibilityProtected,
} LLVMGoVisibility;
typedef enum {
	LLVMGoDLLStorageDefault,
	LLVMGoDLLStorageImport,
	LLVMGoDLLStorageExport,
} LLVMGoDLLStorage;
typedef struct {
	LLVMGoLinkage linkage;
	LLVMGoUnnamedAddr unnamedAddr;
	LLVMGoVisibility visibility;
	LLVMGoDLLStorage dllStorage;
	bool isDSOLocal;
} LLVMGoLinkConfig;
typedef enum {
	LLVMGoTLSNone,
	LLVMGoTLSGeneralDynamic,
	LLVMGoTLSLocalDynamic,
	LLVMGoTLSInitialExec,
	LLVMGoTLSLocalExec,
} LLVMGoTLS;
typedef struct {
	bool isConstant;
	unsigned addrSpace;
	LLVMGoTLS tls;
	LLVMGoLinkConfig link;
	LLVMGoAttributeSetRef attrs;
	bool externallyInitialized;
} LLVMGoVariableOptions;
LLVMValueRef LLVMGoCreateGlobal(
	LLVMModuleRef mod,
	LLVMGoStringRef name,
	LLVMValueRef initializer,
	LLVMGoVariableOptions options
);
LLVMValueRef LLVMGoCreateExternalGlobal(
	LLVMModuleRef mod,
	LLVMGoStringRef name,
	LLVMTypeRef type,
	LLVMGoVariableOptions options
);

// NOTE: LLVMCreateStringAttribute exists, but it uses unsigned int instead of size_t for length.
// (cont): Use our own function to avoid bizzarre overflow edge-cases.
LLVMAttributeRef LLVMGoCreateStringAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef key,
	LLVMGoStringRef value
);
// NOTE: The C API accepts an attribute kind ID when creating a non-string attribute.
// (cont): It is easier to perform the string to ID conversion when creating the attribute.
// (cont): This completely avoids the issue of exposing attribute IDs.
LLVMAttributeRef LLVMGoCreateEnumAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind
);
LLVMAttributeRef LLVMGoCreateIntAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind,
	uint64_t value
);
LLVMAttributeRef LLVMGoCreateTypeAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind,
	LLVMTypeRef value
);
LLVMAttributeRef LLVMGoCreateRangeAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind,
	unsigned bits,
	uint64_t* lowerData,
	size_t lowerLen,
	uint64_t* upperData,
	size_t upperLen
);
bool LLVMGoAttributeKind(LLVMAttributeRef attr, LLVMGoStringRef* key);
bool LLVMGoAttributeStringValue(LLVMAttributeRef attr, LLVMGoStringRef* dst);
bool LLVMGoAttributeIntValue(LLVMAttributeRef attr, uint64_t* dst);
LLVMTypeRef LLVMGoAttributeTypeValue(LLVMAttributeRef attr);
typedef struct {
	uint32_t bits;
	const uint64_t* lower;
	const uint64_t* upper;
} LLVMGoConstRange;
LLVMGoConstRange LLVMGoAttributeRangeValue(LLVMAttributeRef attr);
LLVMGoAttributeSetRef LLVMGoAttributeSetCreate(
	LLVMContextRef ctx,
	LLVMAttributeRef* attrs,
	size_t len
);
LLVMAttributeRef LLVMGoAttributeSetGetString(LLVMGoAttributeSetRef set, LLVMGoStringRef key);
bool LLVMGoAttributeSetGetStringValue(LLVMGoAttributeSetRef set, LLVMGoStringRef key, LLVMGoStringRef* dst);
LLVMAttributeRef LLVMGoAttributeSetGet(LLVMGoAttributeSetRef set, LLVMGoStringRef kind);
bool LLVMGoAttributeSetHasEnum(LLVMGoAttributeSetRef set, LLVMGoStringRef kind);
bool LLVMGoAttributeSetGetInt(LLVMGoAttributeSetRef set, LLVMGoStringRef kind, uint64_t* dst);
LLVMTypeRef LLVMGoAttributeSetGetType(LLVMGoAttributeSetRef set, LLVMGoStringRef kind);
LLVMGoConstRange LLVMGoAttributeSetGetRange(LLVMGoAttributeSetRef set, LLVMGoStringRef kind);
LLVMGoAttributeSetRef LLVMGoAttributeSetMerge(
	LLVMContextRef ctx,
	LLVMGoAttributeSetRef* sets,
	size_t len
);
typedef struct {
	LLVMGoAttributeSetRef result;
	bool ok;
} LLVMGoAttributeSetIntersectResult;
LLVMGoAttributeSetIntersectResult LLVMGoAttributeSetIntersect(
	LLVMContextRef ctx,
	LLVMGoAttributeSetRef first,
	LLVMGoAttributeSetRef* more,
	size_t len
);
typedef enum {
	LLVMGoCaptureAddressFull,
	LLVMGoCaptureAddressIsNull,
	LLVMGoCaptureAddressNone,
} LLVMGoCaptureAddress;
typedef enum {
	LLVMGoCaptureProvenanceFull,
	LLVMGoCaptureProvenanceReadOnly,
	LLVMGoCaptureProvenanceNone,
} LLVMGoCaptureProvenance;
typedef struct {
	LLVMGoCaptureAddress address;
	LLVMGoCaptureProvenance provenance;
} LLVMGoCaptureComponents;
typedef struct {
	LLVMGoCaptureComponents other;
	LLVMGoCaptureComponents returned;
} LLVMGoCaptureInfo;
LLVMGoAttributeSetRef LLVMGoCreateCaptureAttributes(LLVMContextRef ctx, LLVMGoCaptureInfo info);
LLVMGoCaptureInfo LLVMGoGetCaptureInfo(LLVMGoAttributeSetRef attrs);
typedef enum {
	LLVMGoMemoryAccessAny = 0,
	LLVMGoMemoryAccessNoRead = 1 << 0,
	LLVMGoMemoryAccessNoWrite = 1 << 1,
	LLVMGoMemoryAccessNone = LLVMGoMemoryAccessNoRead | LLVMGoMemoryAccessNoWrite,
} LLVMGoMemoryAccessFlags;
#define LLVMGoMemoryAccessFlagsBits 2
typedef enum {
	LLVMGoMemoryLocationOther,
	LLVMGoMemoryLocationArguments,
	LLVMGoMemoryLocationInaccessible,
	LLVMGoMemoryLocationErrno,
	// TODO: add target_mem* when we add support for LLVM 22 (widening LLVMGoMemoryEffectsMask)
} LLVMGoMemoryLocation;
#define LLVMGoMemoryEffectsBroadcast 0x55
typedef uint8_t LLVMGoMemoryEffectsMask;
LLVMGoAttributeSetRef LLVMGoCreateMemoryEffectsAttributes(LLVMContextRef ctx, LLVMGoMemoryEffectsMask mask);
LLVMGoMemoryEffectsMask LLVMGoGetMemoryEffects(LLVMGoAttributeSetRef set);
// Attribute lists cannot be passed directly.
// The context instead owns opaque pointers.
LLVMGoAttributeListRef LLVMGoAttributeListCreate(
	LLVMContextRef ctx,
	LLVMGoAttributeSetRef functionAttributes,
	LLVMGoAttributeSetRef returnAttributes,
	LLVMGoAttributeSetRef* argumentAttributes,
	unsigned argumentsLen
);
LLVMGoAttributeSetRef LLVMGoAttibuteListGetReturn(LLVMGoAttributeListRef list);
LLVMGoAttributeSetRef LLVMGoAttibuteListGetFunction(LLVMGoAttributeListRef list);
LLVMGoAttributeSetRef LLVMGoAttibuteListGetArgument(LLVMGoAttributeListRef list, unsigned index);
unsigned LLVMGoAttributeListArguments(LLVMGoAttributeListRef list);

LLVMBasicBlockRef LLVMGoAppendBasicBlock(LLVMValueRef fn, LLVMGoStringRef name);

LLVMBasicBlockRef LLVMGoAppendBasicBlockAfter(LLVMBasicBlockRef prev, LLVMGoStringRef name);

// Instruction building
// Integer conversions
LLVMValueRef LLVMGoCreateTrunc(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
);
LLVMValueRef LLVMGoCreateZExt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name,
	bool notNegative
);
LLVMValueRef LLVMGoCreateSExt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
// Integer artithmetic
LLVMValueRef LLVMGoCreateAdd(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
);
LLVMValueRef LLVMGoCreateSub(
	LLVMBuilderRef builder,
	LLVMValueRef minuend,
	LLVMValueRef subtrahend,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
);
LLVMValueRef LLVMGoCreateMul(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
);
LLVMValueRef LLVMGoCreateUDiv(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name,
	bool exact
);
LLVMValueRef LLVMGoCreateSDiv(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name,
	bool exact
);
LLVMValueRef LLVMGoCreateURem(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateSRem(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
);
typedef enum {
	LLVMGoIntEQ,
	LLVMGoIntNE,
	LLVMGoIntUGT,
	LLVMGoIntUGE,
	LLVMGoIntULT,
	LLVMGoIntULE,
	LLVMGoIntSGT,
	LLVMGoIntSGE,
	LLVMGoIntSLT,
	LLVMGoIntSLE,
} LLVMGoIntComparison;
LLVMValueRef LLVMGoCreateICmp(
	LLVMBuilderRef builder,
	LLVMGoIntComparison cmp,
	LLVMValueRef lhs,
	LLVMValueRef rhs,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateUMin(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
);
LLVMValueRef LLVMGoCreateSMin(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
);
LLVMValueRef LLVMGoCreateUMax(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
);
LLVMValueRef LLVMGoCreateSMax(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
);
// Bitwise operations
LLVMValueRef LLVMGoCreateShl(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMValueRef by,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
);
LLVMValueRef LLVMGoCreateLShr(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMValueRef by,
	LLVMGoStringRef name,
	bool exact
);
LLVMValueRef LLVMGoCreateAShr(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMValueRef by,
	LLVMGoStringRef name,
	bool exact
);
LLVMValueRef LLVMGoCreateAnd(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateOr(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name,
	bool disjoint
);
LLVMValueRef LLVMGoCreateXOr(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
);
// Special bitwise operations
LLVMValueRef LLVMGoCreateBitReverse(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateBSwap(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateCtPop(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateCtLZ(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name,
	bool nonZero
);
LLVMValueRef LLVMGoCreateCtTZ(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name,
	bool nonZero
);
LLVMValueRef LLVMGoCreateFShl(
	LLVMBuilderRef builder,
	LLVMValueRef high,
	LLVMValueRef low,
	LLVMValueRef by,
	LLVMGoStringRef name
);
// Floating-point casts
LLVMValueRef LLVMGoCreateFPTrunc(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFPExt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateUIToFP(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name,
	bool notNegative
);
LLVMValueRef LLVMGoCreateSIToFP(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFPToUISat(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFPToSISat(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
// Floating-point arithmetic
LLVMValueRef LLVMGoCreateFNeg(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFAdd(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFSub(
	LLVMBuilderRef builder,
	LLVMValueRef minuend,
	LLVMValueRef subtrahend,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFMul(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFDiv(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFRem(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
);
#define LLVMGoFloatEqual (1 << 0)
#define LLVMGoFloatGreater (1 << 1)
#define LLVMGoFloatLess (1 << 2)
#define LLVMGoFloatNaN (1 << 3)
LLVMValueRef LLVMGoCreateFCmp(
	LLVMBuilderRef builder,
	uint8_t cmp,
	LLVMValueRef lhs,
	LLVMValueRef rhs,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateMinimum(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
);
LLVMValueRef LLVMGoCreateMaximum(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
);
// Pointer conversions
LLVMValueRef LLVMGoCreatePtrToInt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateIntToPtr(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateBitCast(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
);
// Memory operations
LLVMValueRef LLVMGoCreateStaticAlloca(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	unsigned addrSpace,
	uint64_t align,
	LLVMGoStringRef name
);
void LLVMGoCreateLifetimeStart(
	LLVMBuilderRef builder,
	LLVMValueRef alloca
);
void LLVMGoCreateLifetimeEnd(
	LLVMBuilderRef builder,
	LLVMValueRef alloca
);
typedef enum {
	LLVMGoMemOrderNormal,
	LLVMGoMemOrderUnordered,
	LLVMGoMemOrderMonotonic,
	LLVMGoMemOrderAcquire,
	LLVMGoMemOrderRelease,
	LLVMGoMemOrderAcquireRelease,
	LLVMGoMemOrderSequentiallyConsistent
} LLVMGoMemOrder;
typedef struct {
	uint64_t align;
	LLVMGoMemOrder order;
	bool isVolatile;
} LLVMGoMemOptions;
LLVMValueRef LLVMGoCreateLoad(
	LLVMBuilderRef builder,
	LLVMTypeRef as,
	LLVMValueRef from,
	LLVMGoMemOptions opts,
	LLVMGoStringRef name
);
void LLVMGoCreateStore(
	LLVMBuilderRef builder,
	LLVMValueRef value,
	LLVMValueRef to,
	LLVMGoMemOptions opts
);
typedef struct {
	uint64_t align;
	bool isVolatile;
	bool isWeak;
} LLVMGoCmpXchgOptions;
LLVMValueRef LLVMGoCreateCmpXchg(
	LLVMBuilderRef builder,
	LLVMValueRef ptr,
	LLVMValueRef from,
	LLVMValueRef to,
	LLVMGoMemOrder success,
	LLVMGoMemOrder failure,
	LLVMGoCmpXchgOptions opts,
	LLVMGoStringRef name
);
typedef enum {
	LLVMGoAtomicRMWOpXchg,
	LLVMGoAtomicRMWOpAdd,
	LLVMGoAtomicRMWOpSub,
	LLVMGoAtomicRMWOpAnd,
	LLVMGoAtomicRMWOpNAnd,
	LLVMGoAtomicRMWOpOr,
	LLVMGoAtomicRMWOpXOr,
	LLVMGoAtomicRMWOpMax,
	LLVMGoAtomicRMWOpMin,
	LLVMGoAtomicRMWOpUMax,
	LLVMGoAtomicRMWOpUMin,
	LLVMGoAtomicRMWOpFAdd,
	LLVMGoAtomicRMWOpFSub,
	LLVMGoAtomicRMWOpFMax,
	LLVMGoAtomicRMWOpFMin,
	// LLVM 16
	LLVMGoAtomicRMWOpUIncWrap,
	LLVMGoAtomicRMWOpUDecWrap,
	// LLVM 20
	LLVMGoAtomicRMWOpUSubCond,
	LLVMGoAtomicRMWOpUSubSat,
	// LLVM 21
	LLVMGoAtomicRMWOpFMaximum,
	LLVMGoAtomicRMWOpFMinimum,
} LLVMGoAtomicRMWOp;
#if LLVM_VERSION_MAJOR >= 21
#define LLVMGoAtomicRMWOpSupported LLVMGoAtomicRMWOpFMinimum
#elif LLVM_VERSION_MAJOR >= 20
#define LLVMGoAtomicRMWOpSupported LLVMGoAtomicRMWOpUSubSat
#elif LLVM_VERSION_MAJOR >= 16
#define LLVMGoAtomicRMWOpSupported LLVMGoAtomicRMWOpUDecWrap
#else
#define LLVMGoAtomicRMWOpSupported LLVMGoAtomicRMWOpFMin
#endif
LLVMValueRef LLVMGoCreateAtomicRMW(
	LLVMBuilderRef builder,
	LLVMGoAtomicRMWOp op,
	LLVMValueRef ptr,
	LLVMValueRef value,
	LLVMGoMemOptions opts,
	LLVMGoStringRef name
);
void LLVMGoCreateMemSet(
	LLVMBuilderRef builder,
	LLVMValueRef dst,
	LLVMValueRef value,
	LLVMValueRef len,
	uint64_t align,
	bool isVolatile
);
void LLVMGoCreateMemCpy(
	LLVMBuilderRef builder,
	LLVMValueRef dst,
	uint64_t dstAlign,
	LLVMValueRef src,
	uint64_t srcAlign,
	LLVMValueRef len,
	bool isVolatile
);
void LLVMGoCreateMemMove(
	LLVMBuilderRef builder,
	LLVMValueRef dst,
	uint64_t dstAlign,
	LLVMValueRef src,
	uint64_t srcAlign,
	LLVMValueRef len,
	bool isVolatile
);
// Aggregate manipulation
LLVMValueRef LLVMGoCreateAggregate(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	LLVMGoStringRef name,
	LLVMValueRef* values,
	unsigned len
);
LLVMValueRef LLVMGoCreateInsertValue(
	LLVMBuilderRef builder,
	LLVMValueRef into,
	LLVMValueRef value,
	unsigned* idxs,
	size_t len,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateExtractValue(
	LLVMBuilderRef builder,
	LLVMValueRef from,
	unsigned* idxs,
	size_t len,
	LLVMGoStringRef name
);
// Control-flow
// Return uses C API
// Jump uses C API
// Branch uses C API
typedef struct {
	LLVMBasicBlockRef ifBlock;
	LLVMBasicBlockRef elseBlock;
} LLVMGoIfBlocks;
LLVMGoIfBlocks LLVMGoCreateIf(
	LLVMBuilderRef builder,
	LLVMValueRef condition,
	LLVMGoStringRef ifName,
	LLVMGoStringRef elseName
);
typedef struct {
	LLVMValueRef index;
	LLVMBasicBlockRef block;
} LLVMGoSwitchCase;
void LLVMGoCreateSwitch(
	LLVMBuilderRef builder,
	LLVMValueRef on,
	LLVMBasicBlockRef defaultBlock,
	LLVMGoSwitchCase* cases,
	size_t len
);
// Unreachable uses C API
typedef struct {
	LLVMValueRef value;
	LLVMBasicBlockRef block;
} LLVMGoPhiIncoming;
LLVMValueRef LLVMGoCreatePhi(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	LLVMGoStringRef name,
	LLVMGoPhiIncoming* incoming,
	size_t len
);
void LLVMGoAddPhiIncoming(
	LLVMValueRef phi,
	LLVMGoPhiIncoming* incoming,
	size_t len
);
LLVMValueRef LLVMGoCreateSelect(
	LLVMBuilderRef builder,
	LLVMValueRef condition,
	LLVMValueRef ifTrue,
	LLVMValueRef ifFalse,
	LLVMGoStringRef name
);

LLVMTypeRef LLVMGoCreateNamedStruct(
	LLVMContextRef ctx,
	LLVMTypeRef* elements,
	size_t elemCount,
	LLVMGoStringRef name
);

// LLVM 17 widened the type of an array's length to uint64_t.
// The signature of the old LLVMArrayType function was left alone for backwards compatability.
// The new LLVMArrayType2 function must be used to provide uint64_t lengths.
// Use #define to select the implementation based on the version.
// https://github.com/llvm/llvm-project/commit/35276f16e5a2cae0dfb49c0fbf874d4d2f177acc
// TODO: Use LLVMArrayType2 directly when we drop LLVM 16 support.
#if LLVM_VERSION_MAJOR >= 17
#define LLVMGoArrayType LLVMArrayType2
#define LLVMGoConstArray LLVMConstArray2
#define LLVMGoArrayLen uint64_t
#else
#define LLVMGoArrayType LLVMArrayType
#define LLVMGoConstArray LLVMConstArray
#define LLVMGoArrayLen unsigned
#endif

typedef enum {
	// Common simple types
	LLVMGoVoidTypeKind,
	LLVMGoIntegerTypeKind,
	LLVMGoPointerTypeKind,

	// Floating-point types
	// Standard IEEE float types
	LLVMGoFloat16Kind,
	LLVMGoFloat32Kind,
	LLVMGoFloat64Kind,
	LLVMGoFloat128Kind,
	// Portable non-standard float types
	LLVMGoBFloat16Kind,
	// Platform-specific float types
	LLVMGoX87Float80Kind,
	LLVMGoPPCFloat128Kind,

	// Aggregate types
	LLVMGoStructTypeKind,
	LLVMGoArrayTypeKind,
	LLVMGoFixedVectorTypeKind,
	LLVMGoScalableVectorTypeKind,

	// Not-directly-usable types
	LLVMGoFunctionTypeKind,
	LLVMGoLabelTypeKind,
	LLVMGoTokenTypeKind,
	LLVMGoMetadataTypeKind,

	// Other types
	// TODO: remove mmx when we drop LLVM 19 support
	// https://github.com/llvm/llvm-project/pull/98505
	LLVMGoX86MMXTypeKind,
	LLVMGoX86AMXTypeKind,
	// NOTE: This is not a normal pointer type, this is a platform-specific hack.
	LLVMGoTypedPointerTypeKind,
	LLVMGoTargetExtensionTypeKind,
} LLVMGoTypeKind;

typedef struct {
	LLVMGoTypeKind kind;
	uint64_t len;
	void* ptr;
} LLVMGoTypeInfo;

LLVMGoTypeInfo LLVMGoGetTypeInfo(LLVMTypeRef t);

LLVMValueRef LLVMGoConstInt(
	LLVMContextRef ctx,
	unsigned bits,
	const uint64_t* data,
	size_t len
);

typedef struct {
	const uint64_t* value;
	unsigned bits;
} LLVMGoIntData;

LLVMGoIntData LLVMGoAsConstInt(LLVMValueRef value);

LLVMValueRef LLVMGoConstIntArray(LLVMContextRef ctx, const char* data, size_t elemSize, size_t len);

LLVMValueRef LLVMGoConstExplicitStruct(LLVMTypeRef type, LLVMValueRef* elements, size_t len);
LLVMValueRef LLVMGoConstImplicitStruct(LLVMContextRef ctx, LLVMValueRef* elements, size_t len);

// Constant expressions

LLVMValueRef LLVMGoConstAdd(
	LLVMValueRef x,
	LLVMValueRef y,
	bool noUnsignedWrap,
	bool noSignedWrap
);
LLVMValueRef LLVMGoConstSub(
	LLVMValueRef minuend,
	LLVMValueRef subtrahend,
	bool noUnsignedWrap,
	bool noSignedWrap
);

// The GEP parameters change between LLVM versions, so use our own.
typedef enum {
	LLVMGoGEPInboundsForwards,
	LLVMGoGEPInbounds,
	LLVMGoGEPForwards,
	LLVMGoGEPSigned,
	LLVMGoGEPUnsigned,
	LLVMGoGEPWrapping,
} LLVMGoGEPMode;

// TODO: add inrange, introduced in LLVM 19
LLVMValueRef LLVMGoConstIndexPointer(
	LLVMTypeRef elemType,
	LLVMValueRef base,
	LLVMValueRef index,
	LLVMGoGEPMode mode
);
LLVMValueRef LLVMGoConstFieldPointer(
	LLVMTypeRef aggType,
	LLVMValueRef base,
	uint32_t index,
	LLVMGoGEPMode mode
);
LLVMValueRef LLVMGoCreateIndexPointer(
	LLVMBuilderRef builder,
	LLVMTypeRef elemType,
	LLVMValueRef base,
	LLVMValueRef index,
	LLVMGoGEPMode mode,
	LLVMGoStringRef name
);
LLVMValueRef LLVMGoCreateFieldPointer(
	LLVMBuilderRef builder,
	LLVMTypeRef aggType,
	LLVMValueRef base,
	uint32_t index,
	LLVMGoGEPMode mode,
	LLVMGoStringRef name
);

// Stringification
void LLVMGoTypeString(void* dst, LLVMTypeRef src);
void LLVMGoValueShortString(void* dst, LLVMValueRef src);
void LLVMGoValueLongString(void* dst, LLVMValueRef src);
void LLVMGoAttributeString(void* dst, LLVMAttributeRef src);
void LLVMGoAttributeSetString(void* dst, LLVMGoAttributeSetRef src);
void LLVMGoAttributeListString(void* dst, LLVMGoAttributeListRef src);
void LLVMGoModuleString(void* dst, LLVMModuleRef src);

#ifdef __cplusplus
}
#endif
