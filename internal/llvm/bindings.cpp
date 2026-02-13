#include "bindings.h"
#include "llvm/IR/LLVMContext.h"
#include "llvm/IR/DerivedTypes.h"
#include "llvm/IR/Constants.h"
#include "llvm/IR/BasicBlock.h"
#include "llvm/IR/Module.h"
#include "llvm/IR/IRBuilder.h"
#include "llvm/MC/TargetRegistry.h"
#include "llvm/Target/TargetMachine.h"

using namespace llvm;

// String conversions
static StringRef toStringRef(LLVMGoStringRef ref) {
	return StringRef(ref.ptr, ref.len);
}
static Twine toTwine(LLVMGoStringRef ref) {
	return Twine(toStringRef(ref));
}
static LLVMGoStringRef fromStringRef(StringRef ref) {
	return {ref.data(), ref.size()};
}
static void LLVMGoConvertString(void* dst, const std::string &src) {
	goCloneString(dst, {src.data(), src.size()});
}

// Misc conversions
static APInt makeAPInt(unsigned bits, const uint64_t* data, size_t len) {
	// A len of 0 is legal from Go's point of view, but not from LLVMs.
	return len != 0 ? APInt(bits, ArrayRef<uint64_t>(data, len)) : APInt(bits, 0);
}

// Context
template <typename T>
struct LLVMGoUniqueRefSet {
	DenseMap<T, std::unique_ptr<T>> set;

	T* wrap(T value) {
		// Map the default value to null.
		if (value == T()) {
			return nullptr;
		}

		// Find or create an entry in the map.
		std::unique_ptr<T> &slot = set[value];
		if (!slot) {
			// Populate the new entry.
			slot.reset(new T(value));
		}

		return slot.get();
	}
};
struct LLVMGoAttributeList {
	AttributeList list;
	LLVMContextRef ctx;
};
struct LLVMGoContext {
	// A struct can be safely pointer-cast to/from its first field type.
	// The context must be first.
	LLVMContext context;

	DenseMap<AttributeSet, std::unique_ptr<AttributeSet>> attrSetRefs;
	DenseMap<AttributeList, std::unique_ptr<LLVMGoAttributeList>> attrListRefs;
};
static LLVMContextRef wrap(LLVMGoContext* ptr) {
	return reinterpret_cast<LLVMContextRef>(ptr);
}
static LLVMGoContext* goUnwrap(LLVMContextRef ref) {
	return reinterpret_cast<LLVMGoContext*>(ref);
}
LLVMContextRef LLVMGoContextCreate() {
	return wrap(new LLVMGoContext());
}
void LLVMGoContextDestroy(LLVMContextRef ctx) {
	delete goUnwrap(ctx);
}

// Target information
static const std::optional<Reloc::Model> LLVMGoRelocationModelsLUT[] = {
	[LLVMGoRelocModelDefault] = std::nullopt,
	[LLVMGoRelocModelStatic] = Reloc::Static,
	[LLVMGoRelocModelPIC] = Reloc::PIC_,
	[LLVMGoRelocModelDynamicNoPIC] = Reloc::DynamicNoPIC,
	[LLVMGoRelocModelROPI] = Reloc::ROPI,
	[LLVMGoRelocModelRWPI] = Reloc::RWPI,
	[LLVMGoRelocModelROPIRWPI] = Reloc::ROPI_RWPI,
};
static const std::optional<CodeModel::Model> LLVMGoCodeModelsLUT[] = {
	[LLVMGoCodeModelDefault] = std::nullopt,
	[LLVMGoCodeModelTiny] = CodeModel::Tiny,
	[LLVMGoCodeModelSmall] = CodeModel::Small,
	[LLVMGoCodeModelKernel] = CodeModel::Kernel,
	[LLVMGoCodeModelMedium] = CodeModel::Medium,
	[LLVMGoCodeModelLarge] = CodeModel::Large,
};
LLVMTargetMachineRef LLVMGoCreateTargetMachine(
	LLVMGoTargetMachineConfig config,
	void* errMsgDst
) {
	StringRef triple = toStringRef(config.triple);

	// Check if the triple is supported.
	// TODO: is there a less cursed way to do this?
	std::string errMsg;
	auto target = TargetRegistry::lookupTarget(triple, errMsg);
	if (target == NULL) {
		// Copy the error string to caller-owned memory.
		LLVMGoConvertString(errMsgDst, errMsg);
		return NULL;
	}

	// Create the target options.
	// TODO: use some of these?
	TargetOptions options = TargetOptions();

	// Create the target machine.
	return reinterpret_cast<LLVMTargetMachineRef>(target->createTargetMachine(
#if LLVM_VERSION_MAJOR >= 21
		Triple(triple),
#else
		triple,
#endif
		toStringRef(config.cpu),
		toStringRef(config.features),
		options,
		LLVMGoRelocationModelsLUT[config.relocationModel],
		LLVMGoCodeModelsLUT[config.codeModel],
		static_cast<CodeGenOptLevel>(config.codeGenLevel)
	));
}
LLVMGoAddressSpaces LLVMGoGetAddressSpaces(LLVMTargetDataRef dataLayout) {
	auto layout = unwrap(dataLayout);
	return {
		.program = layout->getProgramAddressSpace(),
		.global = layout->getDefaultGlobalsAddressSpace(),
		.alloca = layout->getAllocaAddrSpace()
	};
}
LLVMGoABIType LLVMGoGetABIType(LLVMTargetDataRef dataLayout, LLVMTypeRef type) {
	auto layout = unwrap(dataLayout);
	auto ty = unwrap(type);
	auto size = layout->getTypeAllocSize(ty);
	return {
		.size = size.getKnownMinValue(),
		.align = layout->getABITypeAlign(ty).value(),
		.scalable = size.isScalable()
	};
}

// Module accessors
LLVMModuleRef LLVMGoNewModule(
	LLVMContextRef ctx,
	LLVMGoStringRef name,
	LLVMGoStringRef triple,
	LLVMTargetDataRef dataLayout
) {
	auto mod = new Module(toStringRef(name), *unwrap(ctx));
#if LLVM_VERSION_MAJOR >= 21
	mod->setTargetTriple(Triple(toTwine(triple)));
#else
	mod->setTargetTriple(toStringRef(triple));
#endif
	mod->setDataLayout(*unwrap(dataLayout));
	return wrap(mod);
}
LLVMValueRef LLVMGoGetNamedValue(LLVMModuleRef mod, LLVMGoStringRef str) {
	return wrap(unwrap(mod)->getNamedValue(toStringRef(str)));
}

// Attributes
LLVMAttributeRef LLVMGoCreateStringAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef key,
	LLVMGoStringRef value
) {
	return wrap(Attribute::get(
		*unwrap(ctx),
		toStringRef(key),
		toStringRef(value)
	));
}
LLVMAttributeRef LLVMGoCreateEnumAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind
) {
	auto k = Attribute::getAttrKindFromName(toStringRef(kind));
	if (!Attribute::isEnumAttrKind(k)) {
		return nullptr;
	}
	return wrap(Attribute::get(*unwrap(ctx), k));
}
LLVMAttributeRef LLVMGoCreateIntAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind,
	uint64_t value
) {
	auto k = Attribute::getAttrKindFromName(toStringRef(kind));
	if (!Attribute::isIntAttrKind(k)) {
		return nullptr;
	}
	return wrap(Attribute::get(*unwrap(ctx), k, value));
}
LLVMAttributeRef LLVMGoCreateTypeAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind,
	LLVMTypeRef value
) {
	auto k = Attribute::getAttrKindFromName(toStringRef(kind));
	if (!Attribute::isTypeAttrKind(k)) {
		return nullptr;
	}
	return wrap(Attribute::get(*unwrap(ctx), k, unwrap(value)));
}
LLVMAttributeRef LLVMGoCreateRangeAttribute(
	LLVMContextRef ctx,
	LLVMGoStringRef kind,
	unsigned bits,
	uint64_t* lowerData,
	size_t lowerLen,
	uint64_t* upperData,
	size_t upperLen
) {
#if LLVM_VERSION_MAJOR >= 19
	auto k = Attribute::getAttrKindFromName(toStringRef(kind));
	if (!Attribute::isConstantRangeAttrKind(k)) {
		return nullptr;
	}
	ConstantRange range(
		makeAPInt(bits, lowerData, lowerLen),
		makeAPInt(bits, upperData, upperLen)
	);
	return wrap(Attribute::get(*unwrap(ctx), k, range));
#else
	return nullptr;
#endif
}
bool LLVMGoAttributeKind(LLVMAttributeRef attr, LLVMGoStringRef* kind) {
	Attribute a = unwrap(attr);
	if (a.isStringAttribute()) {
		*kind = fromStringRef(a.getKindAsString());
		return true;
	} else {
		*kind = fromStringRef(Attribute::getNameFromAttrKind(a.getKindAsEnum()));
		return false;
	}
}
bool LLVMGoAttributeStringValue(LLVMAttributeRef attr, LLVMGoStringRef* dst) {
	Attribute a = unwrap(attr);
	if (!a.isStringAttribute()) {
		return false;
	}
	*dst = fromStringRef(a.getValueAsString());
	return true;
}
bool LLVMGoAttributeIntValue(LLVMAttributeRef attr, uint64_t* dst) {
	Attribute a = unwrap(attr);
	if (!a.isIntAttribute()) {
		return false;
	}
	*dst = a.getValueAsInt();
	return true;
}
LLVMTypeRef LLVMGoAttributeTypeValue(LLVMAttributeRef attr) {
	Attribute a = unwrap(attr);
	return a.isTypeAttribute() ? wrap(a.getValueAsType()) : nullptr;
}
LLVMGoConstRange LLVMGoAttributeRangeValue(LLVMAttributeRef attr) {
#if LLVM_VERSION_MAJOR >= 19
	Attribute a = unwrap(attr);
	if (!a.isConstantRangeAttribute()) {
		return {0, nullptr, nullptr};
	}
	const ConstantRange& range = a.getValueAsConstantRange();
	return {
		range.getBitWidth(),
		range.getLower().getRawData(),
		range.getUpper().getRawData()
	};
#else
	return {0, nullptr, nullptr};
#endif
}
// Attribute sets
static LLVMGoAttributeSetRef goWrap(LLVMContextRef ctx, AttributeSet set) {
	// Map the empty set to null.
	if (set == AttributeSet()) {
		return nullptr;
	}

	// Find or create an entry in the map.
	std::unique_ptr<AttributeSet> &slot = goUnwrap(ctx)->attrSetRefs[set];
	if (!slot) {
		// Populate the new entry.
		slot.reset(new AttributeSet(set));
	}

	return reinterpret_cast<LLVMGoAttributeSetRef>(slot.get());
}
static LLVMGoAttributeSetRef makeAttrSetRef(LLVMContextRef ctx, const AttrBuilder &builder) {
	return goWrap(ctx, AttributeSet::get(*unwrap(ctx), builder));
}
static AttributeSet unwrap(LLVMGoAttributeSetRef ref) {
	auto ptr = reinterpret_cast<AttributeSet*>(ref);
	return ptr != nullptr ? *ptr : AttributeSet();
}
LLVMGoAttributeSetRef LLVMGoAttributeSetCreate(
	LLVMContextRef ctx,
	LLVMAttributeRef* attrs,
	size_t len
) {
	AttrBuilder builder(*unwrap(ctx));
	for (size_t i = 0; i < len; i++) {
		builder.addAttribute(unwrap(attrs[i]));
	}
	return makeAttrSetRef(ctx, builder);
}
LLVMAttributeRef LLVMGoAttributeSetGetString(LLVMGoAttributeSetRef set, LLVMGoStringRef key) {
	return wrap(unwrap(set).getAttribute(toStringRef(key)));
}
bool LLVMGoAttributeSetGetStringValue(LLVMGoAttributeSetRef set, LLVMGoStringRef key, LLVMGoStringRef* dst) {
	return LLVMGoAttributeStringValue(LLVMGoAttributeSetGetString(set, key), dst);
}
LLVMAttributeRef LLVMGoAttributeSetGet(LLVMGoAttributeSetRef set, LLVMGoStringRef kind) {
	return wrap(unwrap(set).getAttribute(Attribute::getAttrKindFromName(toStringRef(kind))));
}
bool LLVMGoAttributeSetHasEnum(LLVMGoAttributeSetRef set, LLVMGoStringRef kind) {
	auto k = Attribute::getAttrKindFromName(toStringRef(kind));
	return Attribute::isEnumAttrKind(k) && unwrap(set).hasAttribute(k);
}
bool LLVMGoAttributeSetGetInt(LLVMGoAttributeSetRef set, LLVMGoStringRef kind, uint64_t* dst) {
	return LLVMGoAttributeIntValue(LLVMGoAttributeSetGet(set, kind), dst);
}
LLVMTypeRef LLVMGoAttributeSetGetType(LLVMGoAttributeSetRef set, LLVMGoStringRef kind) {
	return LLVMGoAttributeTypeValue(LLVMGoAttributeSetGet(set, kind));
}
LLVMGoConstRange LLVMGoAttributeSetGetRange(LLVMGoAttributeSetRef set, LLVMGoStringRef kind) {
	return LLVMGoAttributeRangeValue(LLVMGoAttributeSetGet(set, kind));
}
LLVMGoAttributeSetRef LLVMGoAttributeSetMerge(
	LLVMContextRef ctx,
	LLVMGoAttributeSetRef* sets,
	size_t len
) {
	LLVMContext* context = unwrap(ctx);
	AttrBuilder builder(*context);
	for (size_t i = 0; i < len; i++) {
		builder.merge(AttrBuilder(*context, unwrap(sets[i])));
	}
	return makeAttrSetRef(ctx, builder);
}
LLVMGoAttributeSetIntersectResult LLVMGoAttributeSetIntersect(
	LLVMContextRef ctx,
	LLVMGoAttributeSetRef first,
	LLVMGoAttributeSetRef* more,
	size_t len
) {
	if (len == 0) {
		return {first, true};
	}
	LLVMContext* context = unwrap(ctx);
	AttributeSet set = unwrap(first);
	for (size_t i = 0; i < len; i++) {
		std::optional<AttributeSet> intersected = set.intersectWith(*context, unwrap(more[i]));
		if (!intersected) {
			return {nullptr, false};
		}
		set = *intersected;
	}
	return {goWrap(ctx, set), true};
}
#if LLVM_VERSION_MAJOR >= 20
static const CaptureComponents LLVMGoCaptureAddressLUT[] = {
	[LLVMGoCaptureAddressFull] = CaptureComponents::Address,
	[LLVMGoCaptureAddressIsNull] = CaptureComponents::AddressIsNull,
	[LLVMGoCaptureAddressNone] = CaptureComponents::None,
};
static const CaptureComponents LLVMGoCaptureProvenanceLUT[] = {
	[LLVMGoCaptureProvenanceFull] = CaptureComponents::Provenance,
	[LLVMGoCaptureProvenanceReadOnly] = CaptureComponents::ReadProvenance,
	[LLVMGoCaptureProvenanceNone] = CaptureComponents::None,
};
static CaptureComponents unwrap(LLVMGoCaptureComponents components) {
	return
		LLVMGoCaptureAddressLUT[components.address] |
		LLVMGoCaptureProvenanceLUT[components.provenance];
}
LLVMGoAttributeSetRef LLVMGoCreateCaptureAttributes(LLVMContextRef ctx, LLVMGoCaptureInfo info) {
	LLVMContext* c = unwrap(ctx);
	CaptureInfo ci(unwrap(info.other), unwrap(info.returned));
#if LLVM_VERSION_MAJOR >= 21
	Attribute attr = Attribute::getWithCaptureInfo(*c, ci);
#else
	// LLVM 20 added a constructor through AttrBuilder but not a normal constructor. 
	// Construct it manually.
	Attribute attr = Attribute::get(*c, Attribute::Captures, ci.toIntValue());
#endif
	return goWrap(ctx, AttributeSet::get(*c, attr));
}
static LLVMGoCaptureComponents goWrap(CaptureComponents components) {
	// TODO: use the captures* check functions once we drop LLVM 20 support
	CaptureComponents address = components & CaptureComponents::Address;
	CaptureComponents provenance = components & CaptureComponents::Provenance;
	return {
		.address = address != CaptureComponents::None
			? address == CaptureComponents::AddressIsNull
				? LLVMGoCaptureAddressIsNull
				: LLVMGoCaptureAddressFull
			: LLVMGoCaptureAddressNone,
		.provenance = provenance != CaptureComponents::None
			? provenance == CaptureComponents::ReadProvenance
				? LLVMGoCaptureProvenanceReadOnly
				: LLVMGoCaptureProvenanceFull
			: LLVMGoCaptureProvenanceNone,
	};
}
LLVMGoCaptureInfo LLVMGoGetCaptureInfo(LLVMGoAttributeSetRef attrs) {
	CaptureInfo info = unwrap(attrs).getCaptureInfo();
	return {
		.other = goWrap(info.getOtherComponents()),
		.returned = goWrap(info.getRetComponents()),
	};
}
#else
LLVMGoAttributeSetRef LLVMGoCreateCaptureAttributes(LLVMContextRef ctx, LLVMGoCaptureInfo info) {
	if (
		info.other.address == LLVMGoCaptureAddressNone &&
		info.other.provenance == LLVMGoCaptureProvenanceNone &&
		info.returned.address == LLVMGoCaptureAddressNone &&
		info.returned.provenance == LLVMGoCaptureProvenanceNone
	) {
		// Create a set with the nocapture attribute.
		Attribute attr = Attribute::get(*c, Attribute::NoCapture);
		return goWrap(ctx, AttributeSet::get(*c, attr));
	}
	// Conservatively omit any capture-related attributes.
	return nullptr;
}
LLVMGoCaptureInfo LLVMGoGetCaptureInfo(LLVMGoAttributeSetRef attrs) {
	CaptureComponents components = unwrap(attrs).hasAttribute(Attribute::NoCapture)
		? {
			.address = LLVMGoCaptureAddressNone,
			.provenance = LLVMGoCaptureProvenanceNone,
		}
		: {
			.address = LLVMGoCaptureAddressFull,
			.provenance = LLVMGoCaptureProvenanceFull,
		};
	return {
		.other = components,
		.returned = components,
	};
}
#endif
// Attribute lists
static LLVMGoAttributeList* unwrap(LLVMGoAttributeListRef ref) {
	return reinterpret_cast<LLVMGoAttributeList*>(ref);
}
static LLVMGoAttributeListRef goWrap(LLVMContextRef ctx, AttributeList list) {
	// Map the empty list value to null.
	if (list == AttributeList()) {
		return nullptr;
	}

	// Find or create an entry in the map.
	std::unique_ptr<LLVMGoAttributeList> &slot = goUnwrap(ctx)->attrListRefs[list];
	if (!slot) {
		// Populate the new entry.
		slot.reset(new LLVMGoAttributeList({list, ctx}));
	}

	return reinterpret_cast<LLVMGoAttributeListRef>(slot.get());
}
LLVMGoAttributeListRef LLVMGoAttributeListCreate(
	LLVMContextRef ctx,
	LLVMGoAttributeSetRef functionAttributes,
	LLVMGoAttributeSetRef returnAttributes,
	LLVMGoAttributeSetRef* argumentAttributes,
	unsigned argumentsLen
) {
	AttributeSet fa = unwrap(functionAttributes);
	AttributeSet ra = unwrap(returnAttributes);
	SmallVector<AttributeSet, 8> aa(argumentsLen);
	for (unsigned i = 0; i < argumentsLen; i++) {
		aa[i] = unwrap(argumentAttributes[i]);
	}
	return goWrap(ctx, AttributeList::get(*unwrap(ctx), fa, ra, aa));
}
static LLVMGoAttributeSetRef LLVMGoAttibuteListGet(LLVMGoAttributeListRef list, unsigned index) {
	if (list == nullptr) {
		return nullptr;
	}
	LLVMGoAttributeList* l = unwrap(list);
	AttributeSet set = l->list.getAttributes(index);
	return goWrap(l->ctx, set);
}
LLVMGoAttributeSetRef LLVMGoAttibuteListGetReturn(LLVMGoAttributeListRef list) {
	return LLVMGoAttibuteListGet(list, AttributeList::ReturnIndex);
}
LLVMGoAttributeSetRef LLVMGoAttibuteListGetFunction(LLVMGoAttributeListRef list) {
	return LLVMGoAttibuteListGet(list, AttributeList::FunctionIndex);
}
LLVMGoAttributeSetRef LLVMGoAttibuteListGetArgument(LLVMGoAttributeListRef list, unsigned index) {
	return LLVMGoAttibuteListGet(list, AttributeList::FirstArgIndex + index);
}
unsigned LLVMGoAttributeListArguments(LLVMGoAttributeListRef list) {
	if (list == nullptr) {
		return 0;
	}
	// The first two entries in the list (if present) are the function attributes and the return attributes.
	// Subtract them from the total set count.
	return std::max(unwrap(list)->list.getNumAttrSets(), 2u) - 2;
}

// Basic-block manipulation
static LLVMBasicBlockRef LLVMGoAppendBasicBlockInner(Function* func, LLVMGoStringRef name, BasicBlock* before) {
	return wrap(BasicBlock::Create(func->getContext(), toTwine(name), func, before));
}
LLVMBasicBlockRef LLVMGoAppendBasicBlock(LLVMValueRef fn, LLVMGoStringRef name) {
	return LLVMGoAppendBasicBlockInner(unwrap<Function>(fn), name, nullptr);
}
LLVMBasicBlockRef LLVMGoAppendBasicBlockAfter(LLVMBasicBlockRef prev, LLVMGoStringRef name) {
	auto prevBlock = unwrap(prev);
	return LLVMGoAppendBasicBlockInner(prevBlock->getParent(), name, prevBlock->getNextNode());
}

// Integer conversions
LLVMValueRef LLVMGoCreateTrunc(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(unwrap(builder)->CreateTrunc(
		unwrap(v),
		unwrap(to),
		toTwine(name),
		noUnsignedWrap,
		noSignedWrap
	));
}
LLVMValueRef LLVMGoCreateZExt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name,
	bool notNegative
) {
	return wrap(unwrap(builder)->CreateZExt(
		unwrap(v),
		unwrap(to),
		toTwine(name)
		// TODO: remove this #if when we drop LLVM 17 support.
#if LLVM_VERSION_MAJOR >= 18
		, notNegative
#endif
	));
}
LLVMValueRef LLVMGoCreateSExt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateSExt(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
// Arithmetic
LLVMValueRef LLVMGoCreateAdd(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(unwrap(builder)->CreateAdd(
		unwrap(x),
		unwrap(y),
		toTwine(name),
		noUnsignedWrap,
		noSignedWrap
	));
}
LLVMValueRef LLVMGoCreateSub(
	LLVMBuilderRef builder,
	LLVMValueRef minuend,
	LLVMValueRef subtrahend,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(unwrap(builder)->CreateSub(
		unwrap(minuend),
		unwrap(subtrahend),
		toTwine(name),
		noUnsignedWrap,
		noSignedWrap
	));
}
LLVMValueRef LLVMGoCreateMul(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(unwrap(builder)->CreateMul(
		unwrap(x),
		unwrap(y),
		toTwine(name),
		noUnsignedWrap,
		noSignedWrap
	));
}
LLVMValueRef LLVMGoCreateUDiv(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name,
	bool exact
) {
	return wrap(unwrap(builder)->CreateUDiv(
		unwrap(dividend),
		unwrap(divisor),
		toTwine(name),
		exact
	));
}
LLVMValueRef LLVMGoCreateSDiv(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name,
	bool exact
) {
	return wrap(unwrap(builder)->CreateSDiv(
		unwrap(dividend),
		unwrap(divisor),
		toTwine(name),
		exact
	));
}
LLVMValueRef LLVMGoCreateURem(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateURem(
		unwrap(dividend),
		unwrap(divisor),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateSRem(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateSRem(
		unwrap(dividend),
		unwrap(divisor),
		toTwine(name)
	));
}
static const ICmpInst::Predicate icmpLUT[] = {
	[LLVMGoIntEQ] = ICmpInst::ICMP_EQ,
	[LLVMGoIntNE] = ICmpInst::ICMP_NE,
	[LLVMGoIntUGT] = ICmpInst::ICMP_UGT,
	[LLVMGoIntUGE] = ICmpInst::ICMP_UGE,
	[LLVMGoIntULT] = ICmpInst::ICMP_ULT,
	[LLVMGoIntULE] = ICmpInst::ICMP_ULE,
	[LLVMGoIntSGT] = ICmpInst::ICMP_SGT,
	[LLVMGoIntSGE] = ICmpInst::ICMP_SGE,
	[LLVMGoIntSLT] = ICmpInst::ICMP_SLT,
	[LLVMGoIntSLE] = ICmpInst::ICMP_SLE,
};
LLVMValueRef LLVMGoCreateICmp(
	LLVMBuilderRef builder,
	LLVMGoIntComparison cmp,
	LLVMValueRef lhs,
	LLVMValueRef rhs,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateICmp(
		icmpLUT[cmp],
		unwrap(lhs),
		unwrap(rhs),
		toTwine(name)
	));
}
static LLVMValueRef LLVMGoIntrinsicReduceList(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len,
	Intrinsic::ID intrinsic
) {
	if (len <= 0) {
		return first;
	}
	auto b = unwrap(builder);
	auto nameTwine = toTwine(name);
	Value* result = unwrap(first);
	for (size_t i = 0; i < len; i++) {
		result = b->CreateBinaryIntrinsic(intrinsic, result, unwrap(more[i]), {}, nameTwine);
	}
	return wrap(result);
}
LLVMValueRef LLVMGoCreateUMin(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
) {
	return LLVMGoIntrinsicReduceList(builder, name, first, more, len, Intrinsic::umin);
}
LLVMValueRef LLVMGoCreateSMin(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
) {
	return LLVMGoIntrinsicReduceList(builder, name, first, more, len, Intrinsic::smin);
}
LLVMValueRef LLVMGoCreateUMax(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
) {
	return LLVMGoIntrinsicReduceList(builder, name, first, more, len, Intrinsic::umax);
}
LLVMValueRef LLVMGoCreateSMax(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
) {
	return LLVMGoIntrinsicReduceList(builder, name, first, more, len, Intrinsic::smax);
}
// Bitwise operations
LLVMValueRef LLVMGoCreateShl(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMValueRef by,
	LLVMGoStringRef name,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(unwrap(builder)->CreateShl(
		unwrap(src),
		unwrap(by),
		toTwine(name),
		noUnsignedWrap,
		noSignedWrap
	));
}
LLVMValueRef LLVMGoCreateLShr(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMValueRef by,
	LLVMGoStringRef name,
	bool exact
) {
	return wrap(unwrap(builder)->CreateLShr(
		unwrap(src),
		unwrap(by),
		toTwine(name),
		exact
	));
}
LLVMValueRef LLVMGoCreateAShr(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMValueRef by,
	LLVMGoStringRef name,
	bool exact
) {
	return wrap(unwrap(builder)->CreateAShr(
		unwrap(src),
		unwrap(by),
		toTwine(name),
		exact
	));
}
LLVMValueRef LLVMGoCreateAnd(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateAnd(
		unwrap(x),
		unwrap(y),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateOr(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name,
	bool disjoint
) {
	return wrap(unwrap(builder)->CreateOr(
		unwrap(x),
		unwrap(y),
		toTwine(name)
#if LLVM_VERSION_MAJOR >= 21
		, disjoint
#else
		// The disjoint flag is not exposed through the CreateOr function until LLVM 21.
		// It is possible to create a disjoint or directly, but not with the folder.
		// Just drop the flag for now.
#endif
	));
}
LLVMValueRef LLVMGoCreateXOr(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateXor(
		unwrap(x),
		unwrap(y),
		toTwine(name)
	));
}
// Special bitwise operations
static LLVMValueRef LLVMGoCreateUnaryIntrinsic(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name,
	Intrinsic::ID intrinsic
) {
	return wrap(unwrap(builder)->CreateUnaryIntrinsic(intrinsic, unwrap(src), {}, toTwine(name)));
}
LLVMValueRef LLVMGoCreateBitReverse(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
) {
	return LLVMGoCreateUnaryIntrinsic(builder, src, name, Intrinsic::bitreverse);
}
LLVMValueRef LLVMGoCreateBSwap(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
) {
	return LLVMGoCreateUnaryIntrinsic(builder, src, name, Intrinsic::bswap);
}
LLVMValueRef LLVMGoCreateCtPop(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
) {
	return LLVMGoCreateUnaryIntrinsic(builder, src, name, Intrinsic::ctpop);
}
static LLVMValueRef LLVMGoCreateBitScanIntrinsic(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name,
	bool nonZero,
	Intrinsic::ID intrinsic
) {
	auto b = unwrap(builder);
	Value* srcVal = unwrap(src);
	return wrap(b->CreateIntrinsic(
		intrinsic,
		unwrap(src)->getType(),
		{srcVal, b->getInt1(nonZero)},
		{},
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateCtLZ(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name,
	bool nonZero
) {
	return LLVMGoCreateBitScanIntrinsic(builder, src, name, nonZero, Intrinsic::ctlz);
}
LLVMValueRef LLVMGoCreateCtTZ(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name,
	bool nonZero
) {
	return LLVMGoCreateBitScanIntrinsic(builder, src, name, nonZero, Intrinsic::cttz);
}
LLVMValueRef LLVMGoCreateFShl(
	LLVMBuilderRef builder,
	LLVMValueRef high,
	LLVMValueRef low,
	LLVMValueRef by,
	LLVMGoStringRef name
) {
	Value* highVal = unwrap(high);
	return wrap(unwrap(builder)->CreateIntrinsic(
		Intrinsic::fshl,
		highVal->getType(),
		{highVal, unwrap(low), unwrap(by)},
		{},
		toTwine(name)
	));
}
// Floating-point casts
LLVMValueRef LLVMGoCreateFPTrunc(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFPTrunc(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFPExt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFPExt(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateUIToFP(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name,
	bool notNegative
) {
	return wrap(unwrap(builder)->CreateUIToFP(
		unwrap(v),
		unwrap(to),
		toTwine(name),
		notNegative
	));
}
LLVMValueRef LLVMGoCreateSIToFP(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateSIToFP(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFPToUISat(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateIntrinsic(
		unwrap(to),
		Intrinsic::fptoui_sat,
		unwrap(v),
		{},
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFPToSISat(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateIntrinsic(
		unwrap(to),
		Intrinsic::fptosi_sat,
		unwrap(v),
		{},
		toTwine(name)
	));
}
// Floating-point arithmetic
LLVMValueRef LLVMGoCreateFNeg(
	LLVMBuilderRef builder,
	LLVMValueRef src,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFNeg(
		unwrap(src),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFAdd(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFAdd(
		unwrap(x),
		unwrap(y),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFSub(
	LLVMBuilderRef builder,
	LLVMValueRef minuend,
	LLVMValueRef subtrahend,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFSub(
		unwrap(minuend),
		unwrap(subtrahend),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFMul(
	LLVMBuilderRef builder,
	LLVMValueRef x,
	LLVMValueRef y,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFMul(
		unwrap(x),
		unwrap(y),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFDiv(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFDiv(
		unwrap(dividend),
		unwrap(divisor),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateFRem(
	LLVMBuilderRef builder,
	LLVMValueRef dividend,
	LLVMValueRef divisor,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFRem(
		unwrap(dividend),
		unwrap(divisor),
		toTwine(name)
	));
}
// These are currently equal, but use a LUT in case LLVM rearranges the fcmp predicates.
static const FCmpInst::Predicate fcmpLUT[] = {
	// 0bNLGE
	[0b0000] = FCmpInst::FCMP_FALSE,
	[0b0001] = FCmpInst::FCMP_OEQ,
	[0b0010] = FCmpInst::FCMP_OGT,
	[0b0011] = FCmpInst::FCMP_OGE,
	[0b0100] = FCmpInst::FCMP_OLT,
	[0b0101] = FCmpInst::FCMP_OLE,
	[0b0110] = FCmpInst::FCMP_ONE,
	[0b0111] = FCmpInst::FCMP_ORD,
	[0b1000] = FCmpInst::FCMP_UNO,
	[0b1001] = FCmpInst::FCMP_UEQ,
	[0b1010] = FCmpInst::FCMP_UGT,
	[0b1011] = FCmpInst::FCMP_UGE,
	[0b1100] = FCmpInst::FCMP_ULT,
	[0b1101] = FCmpInst::FCMP_ULE,
	[0b1110] = FCmpInst::FCMP_UNE,
	[0b1111] = FCmpInst::FCMP_TRUE,
};
LLVMValueRef LLVMGoCreateFCmp(
	LLVMBuilderRef builder,
	uint8_t cmp,
	LLVMValueRef lhs,
	LLVMValueRef rhs,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateFCmp(
		fcmpLUT[cmp],
		unwrap(lhs),
		unwrap(rhs),
		toTwine(name)
	));
}
static LLVMValueRef LLVMGoFloatIntrinsicReduceList(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len,
	Intrinsic::ID unconstrainedIntrinsic,
	Intrinsic::ID constrainedIntrinsic
) {
	if (len <= 0) {
		return first;
	}
	auto b = unwrap(builder);
	auto nameTwine = toTwine(name);
	Value* result = unwrap(first);
	if (b->getIsFPConstrained()) {
		for (size_t i = 0; i < len; i++) {
			result = b->CreateConstrainedFPUnroundedBinOp(
				constrainedIntrinsic,
				result,
				unwrap(more[i]),
				{},
				nameTwine,
				nullptr,
				std::nullopt
			);
		}
	} else {
		for (size_t i = 0; i < len; i++) {
			result = b->CreateBinaryIntrinsic(
				unconstrainedIntrinsic,
				result,
				unwrap(more[i]),
				{},
				nameTwine
			);
		}
	}
	return wrap(result);
}
LLVMValueRef LLVMGoCreateMinimum(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
) {
	return LLVMGoFloatIntrinsicReduceList(
		builder,
		name,
		first,
		more,
		len,
		Intrinsic::minimum,
		Intrinsic::experimental_constrained_minimum
	);
}
LLVMValueRef LLVMGoCreateMaximum(
	LLVMBuilderRef builder,
	LLVMGoStringRef name,
	LLVMValueRef first,
	LLVMValueRef* more,
	size_t len
) {
	return LLVMGoFloatIntrinsicReduceList(
		builder,
		name,
		first,
		more,
		len,
		Intrinsic::maximum,
		Intrinsic::experimental_constrained_maximum
	);
}
// Pointer conversions
LLVMValueRef LLVMGoCreatePtrToInt(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreatePtrToInt(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateIntToPtr(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateIntToPtr(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateBitCast(
	LLVMBuilderRef builder,
	LLVMValueRef v,
	LLVMTypeRef to,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateBitCast(
		unwrap(v),
		unwrap(to),
		toTwine(name)
	));
}
// Memory operations
static Align LLVMGoResolveAlignment(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	uint64_t align
) {
	if (align != 0) {
		return Align(align);
	}
	return unwrap(builder)->GetInsertBlock()->getDataLayout().getABITypeAlign(unwrap(type));
}
LLVMValueRef LLVMGoCreateStaticAlloca(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	unsigned addrSpace,
	uint64_t align,
	LLVMGoStringRef name
) {
	// Create an alloca instruction in the entry block.
	// LLVM does not provide any O(1) method to find the end of the static alloca list, so just insert at the start.
	auto b = unwrap(builder);
	auto inst = new AllocaInst(
		unwrap(type),
		addrSpace,
		nullptr, // omit the "size" option - this is fixed-size
		LLVMGoResolveAlignment(builder, type, align),
		toTwine(name),
		b->GetInsertBlock()->getParent()->getEntryBlock().begin()
	);

	// We did not use the builder to insert the instruction.
	// We need to attach the metadata (e.g. debug info) ourselves.
	b->AddMetadataToInst(inst);

	return wrap(inst);
}
static ConstantInt* LLVMGoAllocaSize(LLVMValueRef alloca) {
	auto a = unwrap<AllocaInst>(alloca);
	auto maybeSize = a->getAllocationSize(a->getParent()->getDataLayout());
	if (!maybeSize) return nullptr;
	auto size = *maybeSize;
	if (!size.isFixed()) return nullptr;
	return ConstantInt::get(a->getContext(), APInt(64, size.getFixedValue()));
}
void LLVMGoCreateLifetimeStart(
	LLVMBuilderRef builder,
	LLVMValueRef alloca
) {
	// LLVM 22 changes the definition of llvm.lifetime.*:
	// - the pointer operand must be a direct reference to an alloca
	// - the size operand is removed
	// We can match LLVM 22's API in older versions by extracting the size from the alloca.
	// TODO: put this in a #if once we add LLVM 22 support.
	unwrap(builder)->CreateLifetimeStart(unwrap(alloca), LLVMGoAllocaSize(alloca));
}
void LLVMGoCreateLifetimeEnd(
	LLVMBuilderRef builder,
	LLVMValueRef alloca
) {
	// LLVM 22 changes the definition of llvm.lifetime.*:
	// - the pointer operand must be a direct reference to an alloca
	// - the size operand is removed
	// We can match LLVM 22's API in older versions by extracting the size from the alloca.
	// TODO: put this in a #if once we add LLVM 22 support.
	unwrap(builder)->CreateLifetimeEnd(unwrap(alloca), LLVMGoAllocaSize(alloca));
}
static const AtomicOrdering LLVMGoMemOrderTable[] = {
	[LLVMGoMemOrderNormal] = AtomicOrdering::NotAtomic,
	[LLVMGoMemOrderUnordered] = AtomicOrdering::Unordered,
	[LLVMGoMemOrderMonotonic] = AtomicOrdering::Monotonic,
	[LLVMGoMemOrderAcquire] = AtomicOrdering::Acquire,
	[LLVMGoMemOrderRelease] = AtomicOrdering::Release,
	[LLVMGoMemOrderAcquireRelease] = AtomicOrdering::AcquireRelease,
	[LLVMGoMemOrderSequentiallyConsistent] = AtomicOrdering::SequentiallyConsistent
};
LLVMValueRef LLVMGoCreateLoad(
	LLVMBuilderRef builder,
	LLVMTypeRef as,
	LLVMValueRef from,
	LLVMGoMemOptions opts,
	LLVMGoStringRef name
) {
	// Directly use the LoadInst constructor to access the full option set.
	return wrap(unwrap(builder)->Insert(
		new LoadInst(
			unwrap(as),
			unwrap(from),
			Twine(),
			opts.isVolatile,
			LLVMGoResolveAlignment(builder, as, opts.align),
			LLVMGoMemOrderTable[opts.order]
		),
		toTwine(name)
	));
}
void LLVMGoCreateStore(
	LLVMBuilderRef builder,
	LLVMValueRef value,
	LLVMValueRef to,
	LLVMGoMemOptions opts
) {
	unwrap(builder)->Insert(new StoreInst(
		unwrap(value),
		unwrap(to),
		opts.isVolatile,
		LLVMGoResolveAlignment(builder, wrap(unwrap(value)->getType()), opts.align),
		LLVMGoMemOrderTable[opts.order]
	));
}
LLVMValueRef LLVMGoCreateCmpXchg(
	LLVMBuilderRef builder,
	LLVMValueRef ptr,
	LLVMValueRef from,
	LLVMValueRef to,
	LLVMGoMemOrder success,
	LLVMGoMemOrder failure,
	LLVMGoCmpXchgOptions opts,
	LLVMGoStringRef name
) {
	// The upstream builder function does not set the name when inserting for some reason?
	// Directly construct and insert it ourselves.
	auto inst = new AtomicCmpXchgInst(
		unwrap(ptr),
		unwrap(from),
		unwrap(to),
		LLVMGoResolveAlignment(builder, wrap(unwrap(from)->getType()), opts.align),
		LLVMGoMemOrderTable[success],
		LLVMGoMemOrderTable[failure],
		SyncScope::System
	);

	// The constructor does not accept the volatile/weak options.
	// Set them seperately.
	inst->setVolatile(opts.isVolatile);
	inst->setWeak(opts.isWeak);

	return wrap(unwrap(builder)->Insert(inst, toTwine(name)));
}
static const AtomicRMWInst::BinOp LLVMGoAtomicRMWOpTable[] = {
	[LLVMGoAtomicRMWOpXchg] = AtomicRMWInst::Xchg,
	[LLVMGoAtomicRMWOpAdd] = AtomicRMWInst::Add,
	[LLVMGoAtomicRMWOpSub] = AtomicRMWInst::Sub,
	[LLVMGoAtomicRMWOpAnd] = AtomicRMWInst::And,
	[LLVMGoAtomicRMWOpNAnd] = AtomicRMWInst::Nand,
	[LLVMGoAtomicRMWOpOr] = AtomicRMWInst::Or,
	[LLVMGoAtomicRMWOpXOr] = AtomicRMWInst::Xor,
	[LLVMGoAtomicRMWOpMax] = AtomicRMWInst::Max,
	[LLVMGoAtomicRMWOpMin] = AtomicRMWInst::Min,
	[LLVMGoAtomicRMWOpUMax] = AtomicRMWInst::UMax,
	[LLVMGoAtomicRMWOpUMin] = AtomicRMWInst::UMin,
	[LLVMGoAtomicRMWOpFAdd] = AtomicRMWInst::FAdd,
	[LLVMGoAtomicRMWOpFSub] = AtomicRMWInst::FSub,
	[LLVMGoAtomicRMWOpFMax] = AtomicRMWInst::FMax,
	[LLVMGoAtomicRMWOpFMin] = AtomicRMWInst::FMin,
#if LLVM_VERSION_MAJOR >= 16
	[LLVMGoAtomicRMWOpUIncWrap] = AtomicRMWInst::UIncWrap,
	[LLVMGoAtomicRMWOpUDecWrap] = AtomicRMWInst::UDecWrap,
#endif
#if LLVM_VERSION_MAJOR >= 20
	[LLVMGoAtomicRMWOpUSubCond] = AtomicRMWInst::USubCond,
	[LLVMGoAtomicRMWOpUSubSat] = AtomicRMWInst::USubSat,
#endif
#if LLVM_VERSION_MAJOR >= 21
	[LLVMGoAtomicRMWOpFMaximum] = AtomicRMWInst::FMaximum,
	[LLVMGoAtomicRMWOpFMinimum] = AtomicRMWInst::FMinimum,
#endif
};
LLVMValueRef LLVMGoCreateAtomicRMW(
	LLVMBuilderRef builder,
	LLVMGoAtomicRMWOp op,
	LLVMValueRef ptr,
	LLVMValueRef value,
	LLVMGoMemOptions opts,
	LLVMGoStringRef name
) {
	// The upstream builder function does not set the name when inserting for some reason?
	// Directly construct and insert it ourselves.
	auto inst = new AtomicRMWInst(
		LLVMGoAtomicRMWOpTable[op],
		unwrap(ptr),
		unwrap(value),
		LLVMGoResolveAlignment(builder, wrap(unwrap(value)->getType()), opts.align),
		LLVMGoMemOrderTable[opts.order],
		SyncScope::System
	);

	// The constructor does not accept the volatile option.
	// Set it seperately.
	inst->setVolatile(opts.isVolatile);

	return wrap(unwrap(builder)->Insert(inst, toTwine(name)));
}
void LLVMGoCreateMemSet(
	LLVMBuilderRef builder,
	LLVMValueRef dst,
	LLVMValueRef value,
	LLVMValueRef len,
	uint64_t align,
	bool isVolatile
) {
	unwrap(builder)->CreateMemSet(
		unwrap(dst),
		unwrap(value),
		unwrap(len),
		MaybeAlign(align),
		isVolatile
	);
}
void LLVMGoCreateMemCpy(
	LLVMBuilderRef builder,
	LLVMValueRef dst,
	uint64_t dstAlign,
	LLVMValueRef src,
	uint64_t srcAlign,
	LLVMValueRef len,
	bool isVolatile
) {
	unwrap(builder)->CreateMemCpy(
		unwrap(dst),
		MaybeAlign(dstAlign),
		unwrap(src),
		MaybeAlign(srcAlign),
		unwrap(len),
		isVolatile
	);
}
void LLVMGoCreateMemMove(
	LLVMBuilderRef builder,
	LLVMValueRef dst,
	uint64_t dstAlign,
	LLVMValueRef src,
	uint64_t srcAlign,
	LLVMValueRef len,
	bool isVolatile
) {
	unwrap(builder)->CreateMemMove(
		unwrap(dst),
		MaybeAlign(dstAlign),
		unwrap(src),
		MaybeAlign(srcAlign),
		unwrap(len),
		isVolatile
	);
}
// Aggregate manipulation
LLVMValueRef LLVMGoCreateAggregate(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	LLVMGoStringRef name,
	LLVMValueRef* values,
	unsigned len
) {
	// Start with a poison value.
	Value* agg = PoisonValue::get(unwrap(type));
	if (len == 0) {
		return wrap(agg);
	}

	// Insert the elements.
	auto b = unwrap(builder);
	auto nameTwine = toTwine(name);
	for (unsigned i = 0; i < len; i++) {
		agg = b->CreateInsertValue(agg, unwrap(values[i]), i, nameTwine);
	}

	return wrap(agg);
}
LLVMValueRef LLVMGoCreateInsertValue(
	LLVMBuilderRef builder,
	LLVMValueRef into,
	LLVMValueRef value,
	unsigned* idxs,
	size_t len,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateInsertValue(
		unwrap(into),
		unwrap(value),
		ArrayRef<unsigned>(idxs, len),
		toTwine(name)
	));
}
LLVMValueRef LLVMGoCreateExtractValue(
	LLVMBuilderRef builder,
	LLVMValueRef from,
	unsigned* idxs,
	size_t len,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateExtractValue(
		unwrap(from),
		ArrayRef<unsigned>(idxs, len),
		toTwine(name)
	));
}
// Control-flow
LLVMGoIfBlocks LLVMGoCreateIf(
	LLVMBuilderRef builder,
	LLVMValueRef condition,
	LLVMGoStringRef ifName,
	LLVMGoStringRef elseName
) {
	// Create two blocks after the current block.
	auto b = unwrap(builder);
	auto brBlock = b->GetInsertBlock();
	auto func = brBlock->getParent();
	auto& ctx = func->getContext();
	auto next = brBlock->getNextNode();
	auto ifBlock = BasicBlock::Create(ctx, toTwine(ifName), func, next);
	auto elseBlock = BasicBlock::Create(ctx, toTwine(elseName), func, next);

	// Create the branch.
	b->CreateCondBr(unwrap(condition), ifBlock, elseBlock);

	// Move the insertion point to the if block.
	b->SetInsertPoint(ifBlock);

	return {wrap(ifBlock), wrap(elseBlock)};
}
void LLVMGoCreateSwitch(
	LLVMBuilderRef builder,
	LLVMValueRef on,
	LLVMBasicBlockRef defaultBlock,
	LLVMGoSwitchCase* cases,
	size_t len
) {
	auto inst = unwrap(builder)->CreateSwitch(
		unwrap(on),
		unwrap(defaultBlock),
		(unsigned)len
	);
	for (size_t i = 0; i < len; i++) {
		inst->addCase(
			unwrap<ConstantInt>(cases[i].index),
			unwrap(cases[i].block)
		);
	}
}
static void LLVMGoAddPhiIncomingInner(
	PHINode* phi,
	LLVMGoPhiIncoming* incoming,
	size_t len
) {
	for (size_t i = 0; i < len; i++) {
		phi->addIncoming(
			unwrap(incoming[i].value),
			unwrap(incoming[i].block)
		);
	}
}
LLVMValueRef LLVMGoCreatePhi(
	LLVMBuilderRef builder,
	LLVMTypeRef type,
	LLVMGoStringRef name,
	LLVMGoPhiIncoming* incoming,
	size_t len
) {
	auto inst = unwrap(builder)->CreatePHI(
		unwrap(type),
		(unsigned)len,
		toTwine(name)
	);
	LLVMGoAddPhiIncomingInner(inst, incoming, len);
	return wrap(inst);
}
void LLVMGoAddPhiIncoming(
	LLVMValueRef phi,
	LLVMGoPhiIncoming* incoming,
	size_t len
) {
	LLVMGoAddPhiIncomingInner(unwrap<PHINode>(phi), incoming, len);
}
LLVMValueRef LLVMGoCreateSelect(
	LLVMBuilderRef builder,
	LLVMValueRef condition,
	LLVMValueRef ifTrue,
	LLVMValueRef ifFalse,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateSelect(
		unwrap(condition),
		unwrap(ifTrue),
		unwrap(ifFalse),
		toTwine(name)
	));
}

LLVMTypeRef LLVMGoCreateNamedStruct(
	LLVMContextRef ctx,
	LLVMTypeRef* elements,
	size_t elemCount,
	LLVMGoStringRef name
) {
	return wrap(StructType::create(
		*unwrap(ctx),
		ArrayRef<Type*>(unwrap(elements), elemCount),
		toStringRef(name),
		false
	));
}

LLVMGoTypeInfo LLVMGoGetTypeInfo(LLVMTypeRef t) {
	Type* ty = unwrap(t);
	switch (ty->getTypeID()) {
	case Type::TypeID::VoidTyID:
		return {LLVMGoVoidTypeKind, 0, NULL};
	case Type::TypeID::HalfTyID:
		return {LLVMGoFloat16Kind, 0, NULL};
	case Type::TypeID::FloatTyID:
		return {LLVMGoFloat32Kind, 0, NULL};
	case Type::TypeID::DoubleTyID:
		return {LLVMGoFloat64Kind, 0, NULL};
	case Type::TypeID::FP128TyID:
		return {LLVMGoFloat128Kind, 0, NULL};
	case Type::TypeID::BFloatTyID:
		return {LLVMGoBFloat16Kind, 0, NULL};
	case Type::TypeID::X86_FP80TyID:
		return {LLVMGoX87Float80Kind, 0, NULL};
	case Type::TypeID::PPC_FP128TyID:
		return {LLVMGoPPCFloat128Kind, 0, NULL};
	case Type::TypeID::FunctionTyID:
		return {LLVMGoFunctionTypeKind, 0, NULL};
	case Type::TypeID::LabelTyID:
		return {LLVMGoLabelTypeKind, 0, NULL};
	case Type::TypeID::TokenTyID:
		return {LLVMGoTokenTypeKind, 0, NULL};
	case Type::TypeID::MetadataTyID:
		return {LLVMGoMetadataTypeKind, 0, NULL};
#if LLVM_VERSION_MAJOR < 20
	// The x86_mmx type was removed in LLVM 20.
	case Type::TypeID::X86_MMXTyID:
		return {LLVMGoX86MMXTypeKind, 0, NULL};
#endif
	case Type::TypeID::X86_AMXTyID:
		return {LLVMGoX86AMXTypeKind, 0, NULL};
#if LLVM_VERSION_MAJOR >= 16
	case Type::TypeID::TypedPointerTyID:
		// This type has more info, but TinyGo does not need it.
		return {LLVMGoTypedPointerTypeKind, 0, NULL};
	case Type::TypeID::TargetExtTyID:
		// This type has more info, but TinyGo does not need it.
		return {LLVMGoTargetExtensionTypeKind, 0, NULL};
#endif
#if LLVM_VERSION_MAJOR == 15
	case Type::TypeID::DXILPointerTyID:
		// This type has more info, but TinyGo does not need it.
		return {LLVMGoTypedPointerTypeKind, 0, NULL};
#endif
	case Type::TypeID::IntegerTyID:
		return {LLVMGoIntegerTypeKind, ty->getIntegerBitWidth(), NULL};
	case Type::TypeID::PointerTyID:
		return {LLVMGoPointerTypeKind, cast<PointerType>(ty)->getAddressSpace(), NULL};
	case Type::TypeID::StructTyID:
		// The lifetime of the ArrayRef requires a seperate scope.
		{
			ArrayRef<Type*> elements = cast<StructType>(ty)->elements();
			return {LLVMGoStructTypeKind, elements.size(), (void*)elements.data()};
		}
	case Type::TypeID::ArrayTyID:
		return {LLVMGoArrayTypeKind, ty->getArrayNumElements(), ty->getArrayElementType()};
	case Type::TypeID::FixedVectorTyID:
		// This type has more info, but TinyGo does not need it.
		return {LLVMGoFixedVectorTypeKind, 0, NULL};
	case Type::TypeID::ScalableVectorTyID:
		// This type has more info, but TinyGo does not need it.
		return {LLVMGoScalableVectorTypeKind, 0, NULL};
	default:
		// TODO: add an error message or something
		abort();
	}
}

LLVMValueRef LLVMGoConstInt(
	LLVMContextRef ctx,
	unsigned bits,
	const uint64_t* data,
	size_t len
) {
	return wrap(ConstantInt::get(*unwrap(ctx), makeAPInt(bits, data, len)));
}

LLVMGoIntData LLVMGoAsConstInt(LLVMValueRef value) {
	if (auto *i = dyn_cast_or_null<ConstantInt>(unwrap(value))) {
		const APInt& val = i->getValue();
		return {val.getRawData(), val.getBitWidth()};
	} else {
		return {NULL, 0};
	}
}

LLVMValueRef LLVMGoConstIntArray(LLVMContextRef ctx, const char* data, size_t elemSize, size_t len) {
	return wrap(ConstantDataArray::getRaw(
		StringRef(data, elemSize*len),
		len,
		IntegerType::get(*unwrap(ctx), 8*elemSize))
	);
}

// LLVM's unwrap-as-constant uses the wrong length type (unsigned instead of size_t), and may explode.
// We must use reinterpret_cast until this is fixed.

LLVMValueRef LLVMGoConstExplicitStruct(LLVMTypeRef type, LLVMValueRef *elements, size_t len) {
	return wrap(ConstantStruct::get(unwrap<StructType>(type), ArrayRef<Constant*>(reinterpret_cast<Constant**>(elements), len)));
}

LLVMValueRef LLVMGoConstImplicitStruct(LLVMContextRef ctx, LLVMValueRef *elements, size_t len) {
	return wrap(ConstantStruct::getAnon(*unwrap(ctx), ArrayRef<Constant*>(reinterpret_cast<Constant**>(elements), len)));
}


// Constant expressions

LLVMValueRef LLVMGoConstAdd(
	LLVMValueRef x,
	LLVMValueRef y,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(ConstantExpr::getAdd(
		unwrap<Constant>(x),
		unwrap<Constant>(y),
		noUnsignedWrap,
		noSignedWrap
	));
}

LLVMValueRef LLVMGoConstSub(
	LLVMValueRef minuend,
	LLVMValueRef subtrahend,
	bool noUnsignedWrap,
	bool noSignedWrap
) {
	return wrap(ConstantExpr::getSub(
		unwrap<Constant>(minuend),
		unwrap<Constant>(subtrahend),
		noUnsignedWrap,
		noSignedWrap
	));
}

// GEP stuff

#if LLVM_VERSION_MAJOR >= 19
static const GEPNoWrapFlags LLVMGoConvertWrapLUT[] = {
	[LLVMGoGEPInboundsForwards] = GEPNoWrapFlags::all(),
	[LLVMGoGEPInbounds] = GEPNoWrapFlags::inBounds(),
	[LLVMGoGEPForwards] = GEPNoWrapFlags::noUnsignedSignedWrap() | GEPNoWrapFlags::noUnsignedWrap(),
	[LLVMGoGEPSigned] = GEPNoWrapFlags::noUnsignedSignedWrap(),
	[LLVMGoGEPUnsigned] = GEPNoWrapFlags::noUnsignedWrap(),
	[LLVMGoGEPWrapping] = GEPNoWrapFlags::none(),
};
static GEPNoWrapFlags LLVMGoConvertWrap(LLVMGoGEPMode mode) {
	return LLVMGoConvertWrapLUT[mode];
}
#else
// There was only an "inbounds" flag prior to LLVM 19.
static bool LLVMGoConvertWrap(LLVMGoGEPMode mode) {
	return mode == LLVMGoGEPInboundsForwards || mode == LLVMGoGEPInbounds;
}
#endif
LLVMValueRef LLVMGoConstIndexPointer(
	LLVMTypeRef elemType,
	LLVMValueRef base,
	LLVMValueRef index,
	LLVMGoGEPMode mode
) {
	return wrap(ConstantExpr::getGetElementPtr(
		unwrap(elemType),
		unwrap<Constant>(base),
		unwrap<Constant>(index),
		LLVMGoConvertWrap(mode)
	));
}
LLVMValueRef LLVMGoConstFieldPointer(
	LLVMTypeRef aggType,
	LLVMValueRef base,
	uint32_t index,
	LLVMGoGEPMode mode
) {
	Type* ty = unwrap(aggType);
	LLVMContext &context = ty->getContext();
	return wrap(ConstantExpr::getGetElementPtr(
		ty,
		unwrap<Constant>(base),
		ArrayRef<Constant*>({
			ConstantInt::get(context, APInt(32, 0)),
			ConstantInt::get(context, APInt(32, index))
		}),
		LLVMGoConvertWrap(mode)
	));
}
LLVMValueRef LLVMGoCreateIndexPointer(
	LLVMBuilderRef builder,
	LLVMTypeRef elemType,
	LLVMValueRef base,
	LLVMValueRef index,
	LLVMGoGEPMode mode,
	LLVMGoStringRef name
) {
	return wrap(unwrap(builder)->CreateGEP(
		unwrap(elemType),
		unwrap(base),
		unwrap(index),
		toTwine(name),
		LLVMGoConvertWrap(mode)
	));
}
LLVMValueRef LLVMGoCreateFieldPointer(
	LLVMBuilderRef builder,
	LLVMTypeRef aggType,
	LLVMValueRef base,
	uint32_t index,
	LLVMGoGEPMode mode,
	LLVMGoStringRef name
) {
	auto b = unwrap(builder);
	return wrap(b->CreateGEP(
		unwrap(aggType),
		unwrap(base),
		{
			b->getInt32(0),
			b->getInt32(index)
		},
		toTwine(name),
		LLVMGoConvertWrap(mode)
	));
}

// Stringification
void LLVMGoTypeString(void* dst, LLVMTypeRef src) {
	std::string str;
	raw_string_ostream stream(str);
	unwrap(src)->print(stream);
	LLVMGoConvertString(dst, str);
}
void LLVMGoValueString(void* dst, LLVMValueRef src) {
	std::string str;
	raw_string_ostream stream(str);
	unwrap(src)->print(stream);
	LLVMGoConvertString(dst, str);
}
void LLVMGoAttributeString(void* dst, LLVMAttributeRef src) {
	std::string str = unwrap(src).getAsString();
	LLVMGoConvertString(dst, str);
}
void LLVMGoAttributeSetString(void* dst, LLVMGoAttributeSetRef src) {
	std::string str = unwrap(src).getAsString();
	LLVMGoConvertString(dst, str);
}
void LLVMGoAttributeListString(void* dst, LLVMGoAttributeListRef src) {
	std::string str;
	raw_string_ostream stream(str);
	AttributeList list;
	if (src != nullptr) {
		list = unwrap(src)->list;
	}
	list.print(stream);
	LLVMGoConvertString(dst, str);
}
void LLVMGoModuleString(void* dst, LLVMModuleRef src) {
	std::string str;
	raw_string_ostream stream(str);
	unwrap(src)->print(stream, nullptr);
	LLVMGoConvertString(dst, str);
}
