import peachpy.x86_64

class State:
    def __init__(self):
        self.v0lo = XMMRegister()
        self.v0hi = XMMRegister()
        self.v1lo = XMMRegister()
        self.v1hi = XMMRegister()
        self.mul0lo = XMMRegister()
        self.mul0hi = XMMRegister()
        self.mul1lo = XMMRegister()
        self.mul1hi = XMMRegister()

    def load(self,ptr):
        # load state into xmm registers
        for i, r in enumerate([self.v0lo, self.v0hi, self.v1lo, self.v1hi, self.mul0lo, self.mul0hi, self.mul1lo, self.mul1hi]):
            MOVDQU(r, [ptr+i*r.size])


    def store(self,ptr):
        # load state into xmm registers
        for i, r in enumerate([self.v0lo, self.v0hi, self.v1lo, self.v1hi, self.mul0lo, self.mul0hi, self.mul1lo, self.mul1hi]):
            MOVDQU([ptr+i*r.size], r)

def mm_shufmask(a,b,c,d): return (a << 6) | (b << 4) | (c << 2) | d

def permute(dstlo,dsthi,srclo,srchi):
        PSHUFD(dstlo, srchi, mm_shufmask(2,3,0,1))
        PSHUFD(dsthi, srclo, mm_shufmask(2,3,0,1))

def MakePermuteSSE():

    vptr = Argument(ptr())
    permuted_ptr = Argument(ptr())

    with Function("permuteSSE", (vptr, permuted_ptr), target=uarch.default) as function:
        reg_vptr = GeneralPurposeRegister64()
        reg_permuted_ptr = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_vptr, vptr)
        LOAD.ARGUMENT(reg_permuted_ptr, permuted_ptr)

        plo, phi = XMMRegister(), XMMRegister()
        vlo, vhi = XMMRegister(), XMMRegister()

        MOVDQU(vhi, [reg_vptr+16])
        MOVDQU(vlo, [reg_vptr])

        permute(plo,phi,vlo,vhi)

        MOVDQU([reg_permuted_ptr], plo)
        MOVDQU([reg_permuted_ptr+16], phi)

        RETURN()


MakePermuteSSE()

def zippermask():
    x = GeneralPurposeRegister64()
    mask = XMMRegister()
    tmpmask = XMMRegister()

    MOV(x, 0x000F010E05020C03)
    MOVQ(mask, x)
    MOV(x, 0x070806090D0A040B)
    MOVQ(tmpmask, x)
    MOVLHPS(mask, tmpmask)

    return mask

def zipper(mask,mlo,mhi,vlo,vhi):
    MOVDQA(vlo,mlo)
    PSHUFB(vlo,mask)
    MOVDQA(vhi,mhi)
    PSHUFB(vhi,mask)

def MakeZipperSSE():
    mul0 = Argument(ptr())
    v0 = Argument(ptr())

    with Function("zipperSSE", (mul0, v0), target=uarch.default+isa.ssse3) as function:
        reg_mul0 = GeneralPurposeRegister64()
        reg_v0 = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_mul0, mul0)
        LOAD.ARGUMENT(reg_v0, v0)

        m0lo, m0hi = XMMRegister(), XMMRegister()
        vlo, vhi = XMMRegister(), XMMRegister()

        MOVDQU(m0lo, [reg_mul0])
        MOVDQU(m0hi, [reg_mul0+16])

        mask = zippermask()
        zipper(mask,m0lo,m0hi,vlo,vhi)

        MOVDQU([reg_v0], vlo)
        MOVDQU([reg_v0+16], vhi)

        RETURN()

MakeZipperSSE()

def update(plo,phi, state):
        PADDQ(state.v1lo, plo)
        PADDQ(state.v1hi, phi)
        PADDQ(state.v1lo, state.mul0lo)
        PADDQ(state.v1hi, state.mul0hi)

        dstlo = XMMRegister()
        dsthi = XMMRegister()
        srclo = XMMRegister()
        srchi = XMMRegister()

        MOVDQA(srclo, state.v0lo)
        MOVDQA(srchi, state.v0hi)
        MOVDQA(dstlo, state.v1lo)
        MOVDQA(dsthi, state.v1hi)
        PSRLQ(dstlo, 32)
        PSRLQ(dsthi, 32)

        PMULUDQ(dstlo, srclo)
        PMULUDQ(dsthi, srchi)
        PXOR(state.mul0lo, dstlo)
        PXOR(state.mul0hi, dsthi)

        ###

        PADDQ(state.v0lo, state.mul1lo)
        PADDQ(state.v0hi, state.mul1hi)

        ###

        MOVDQA(srclo, state.v1lo)
        MOVDQA(srchi, state.v1hi)
        MOVDQA(dstlo, state.v0lo)
        MOVDQA(dsthi, state.v0hi)
        PSRLQ(dstlo, 32)
        PSRLQ(dsthi, 32)

        PMULUDQ(dstlo, srclo)
        PMULUDQ(dsthi, srchi)
        PXOR(state.mul1lo, dstlo)
        PXOR(state.mul1hi, dsthi)

        ######

        mask = zippermask()
        zipper(mask, state.v1lo, state.v1hi, dstlo, dsthi)
        PADDQ(state.v0lo, dstlo)
        PADDQ(state.v0hi, dsthi)

        zipper(mask, state.v0lo, state.v0hi, dstlo, dsthi)
        PADDQ(state.v1lo, dstlo)
        PADDQ(state.v1hi, dsthi)


def MakeUpdateSSE():

    sptr = Argument(ptr())
    p_base = Argument(ptr())
    p_len = Argument(int64_t)
    p_cap = Argument(int64_t)

    with Function("updateSSE", (sptr, p_base, p_len, p_cap), target=uarch.default + isa.ssse3) as function:
        reg_sptr = GeneralPurposeRegister64()
        reg_p = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_sptr, sptr)
        LOAD.ARGUMENT(reg_p, p_base)

        state = State()

        state.load(reg_sptr)

        reg_plo = XMMRegister()
        reg_phi = XMMRegister()

        MOVDQU(reg_plo, [reg_p])
        MOVDQU(reg_phi, [reg_p+16])

        update(reg_plo, reg_phi, state)

        state.store(reg_sptr)

        RETURN()

MakeUpdateSSE()

def MakeUpdateStateSSE():

    sptr = Argument(ptr())
    p_base = Argument(ptr())
    p_len = Argument(int64_t)
    p_cap = Argument(int64_t)

    with Function("updateStateSSE", (sptr, p_base, p_len, p_cap), target=uarch.default + isa.ssse3) as function:
        reg_sptr = GeneralPurposeRegister64()
        reg_p = GeneralPurposeRegister64()
        reg_p_len = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_sptr, sptr)
        LOAD.ARGUMENT(reg_p, p_base)
        LOAD.ARGUMENT(reg_p_len, p_len)

        state = State()

        state.load(reg_sptr)

        reg_plo = XMMRegister()
        reg_phi = XMMRegister()

        loop = Loop()
        CMP(reg_p_len, 0)
        JE(loop.end)
        with loop:

            MOVDQU(reg_plo, [reg_p])
            MOVDQU(reg_phi, [reg_p+16])

            update(reg_plo, reg_phi, state)

            ADD(reg_p, 32)
            SUB(reg_p_len, 32)
            CMP(reg_p_len, 0)
            JNE(loop.begin)


        state.store(reg_sptr)

        RETURN()

MakeUpdateStateSSE()

def permuteAndUpdate(state):
    plo, phi = XMMRegister(), XMMRegister()

    permute(plo,phi,state.v0lo,state.v0hi)
    update(plo,phi,state)

def MakePermuteAndUpdate():

    sptr = Argument(ptr())

    with Function("permuteAndUpdateSSE", (sptr,), target=uarch.default + isa.ssse3) as function:
        reg_sptr = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_sptr, sptr)

        state = State()

        state.load(reg_sptr)

        permuteAndUpdate(state)

        state.store(reg_sptr)

        RETURN()

MakePermuteAndUpdate()

def MakeFinalize():

    sptr = Argument(ptr())

    with Function("finalizeSSE", (sptr,), uint64_t, target=uarch.default + isa.ssse3) as function:
        reg_sptr = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_sptr, sptr)

        state = State()

        state.load(reg_sptr)

        c = GeneralPurposeRegister64()
        MOV(c, 4)
        with Loop() as loop:
            permuteAndUpdate(state)
            DEC(c)
            JNZ(loop.begin)

        PADDQ(state.v0lo, state.v1lo)
        PADDQ(state.mul0lo, state.mul1lo)

        PADDQ(state.v0lo, state.mul0lo)

        ret = GeneralPurposeRegister64()

        MOVQ(ret, state.v0lo)

        RETURN(ret)

MakeFinalize()
