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

    def load(self, ptr):
        # load state into xmm registers
        for i, r in enumerate([
                self.v0lo, self.v0hi, self.v1lo, self.v1hi, self.mul0lo,
                self.mul0hi, self.mul1lo, self.mul1hi
        ]):
            MOVDQU(r, [ptr + i * r.size])

    def store(self, ptr):
        # load state into xmm registers
        for i, r in enumerate([
                self.v0lo, self.v0hi, self.v1lo, self.v1hi, self.mul0lo,
                self.mul0hi, self.mul1lo, self.mul1hi
        ]):
            MOVDQU([ptr + i * r.size], r)


def mm_shufmask(a, b, c, d):
    return (a << 6) | (b << 4) | (c << 2) | d


def permute(dstlo, dsthi, srclo, srchi):
    PSHUFD(dstlo, srchi, mm_shufmask(2, 3, 0, 1))
    PSHUFD(dsthi, srclo, mm_shufmask(2, 3, 0, 1))

def rotate32By(dst, count):
    t = XMMRegister()
    c = XMMRegister()
    MOVDQA(t, dst)
    MOVQ(c, count)
    PSLLD(t, c)
    SUB(count, 32)
    NEG(count)
    MOVQ(c, count)
    PSRLD(dst, c)
    POR(dst, t)

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


def zipper(mask, mlo, mhi, vlo, vhi):
    MOVDQA(vlo, mlo)
    PSHUFB(vlo, mask)
    MOVDQA(vhi, mhi)
    PSHUFB(vhi, mask)


def update(plo, phi, state):
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
    PSRLQ(srclo, 32)
    PSRLQ(srchi, 32)

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
    PSRLQ(srclo, 32)
    PSRLQ(srchi, 32)

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


def permuteAndUpdate(state):
    plo, phi = XMMRegister(), XMMRegister()

    permute(plo, phi, state.v0lo, state.v0hi)
    update(plo, phi, state)


def finalize(state):
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

    return ret


def newstate(reg_keys, reg_init0, reg_init1):
    state = State()

    MOVDQU(state.v0lo, [reg_keys])
    MOVDQU(state.v0hi, [reg_keys + 16])
    MOVDQU(state.mul0lo, [reg_init0])
    MOVDQU(state.mul0hi, [reg_init0 + 16])
    MOVDQU(state.mul1lo, [reg_init1])
    MOVDQU(state.mul1hi, [reg_init1 + 16])

    permute(state.v1hi, state.v1lo, state.v0lo, state.v0hi)

    PXOR(state.v0lo, state.mul0lo)
    PXOR(state.v0hi, state.mul0hi)
    PXOR(state.v1lo, state.mul1lo)
    PXOR(state.v1hi, state.mul1hi)

    return state


def memcpy32(x0, x1, p, l):

    fin = Label("memcpy32_fin")
    CMP(l, 0)
    JE(fin)

    skipLoad16 = Label("memcpy32_skipLoad16")
    CMP(l, 16)
    JL(skipLoad16)
    MOVDQU(x0, [p])
    ADD(p, 16)
    SUB(l, 16)
    memcpy16(x1, p, l)
    JMP(fin)
    LABEL(skipLoad16)
    memcpy16(x0, p, l)

    LABEL(fin)


def memcpy16(xmm0, p, l):

    b = GeneralPurposeRegister64()
    offs = GeneralPurposeRegister64()
    XOR(offs, offs)

    skip8 = Label()
    CMP(l, 8)
    JL(skip8)
    MOV(b, [p])
    MOVQ(xmm0, b)
    SUB(l, 8)
    ADD(p, 8)
    MOV(offs, 1)
    LABEL(skip8)

    XOR(b, b)
    # no support for jump tables
    labels = [Label() for i in range(0, 8)]
    for i in range(0, 7):
        CMP(l, i)
        JE(labels[i])
    char = GeneralPurposeRegister64()
    for i in range(7, 0, -1):
        LABEL(labels[i])
        MOVZX(char, byte[p + i - 1])
        SHL(char, (i - 1) * 8)
        OR(b, char)

    fin16 = Label()
    insert1 = Label()
    CMP(offs, 1)
    JZ(insert1)
    PINSRQ(xmm0, b, 0)
    JMP(fin16)
    LABEL(insert1)
    PINSRQ(xmm0, b, 1)
    LABEL(fin16)
    LABEL(labels[0])


def MakeHash():

    keys = Argument(ptr())
    init0 = Argument(ptr())
    init1 = Argument(ptr())
    p_base = Argument(ptr())
    p_len = Argument(int64_t)
    p_cap = Argument(int64_t)

    with Function(
            "hashSSE", (keys, init0, init1, p_base, p_len, p_cap),
            uint64_t,
            target=uarch.default + isa.sse4_1) as function:

        reg_keys = GeneralPurposeRegister64()
        reg_init0 = GeneralPurposeRegister64()
        reg_init1 = GeneralPurposeRegister64()

        LOAD.ARGUMENT(reg_keys, keys)
        LOAD.ARGUMENT(reg_init0, init0)
        LOAD.ARGUMENT(reg_init1, init1)
        state = newstate(reg_keys, reg_init0, reg_init1)

        reg_p = GeneralPurposeRegister64()
        reg_p_len = GeneralPurposeRegister64()
        LOAD.ARGUMENT(reg_p, p_base)
        LOAD.ARGUMENT(reg_p_len, p_len)

        reg_plo = XMMRegister()
        reg_phi = XMMRegister()

        loop = Loop()
        CMP(reg_p_len, 32)
        JL(loop.end)
        with loop:
            MOVDQU(reg_plo, [reg_p])
            MOVDQU(reg_phi, [reg_p + 16])

            update(reg_plo, reg_phi, state)

            ADD(reg_p, 32)
            SUB(reg_p_len, 32)
            CMP(reg_p_len, 32)
            JGE(loop.begin)

        ###

        # reg_p_len is now remainder mod 32
        lfinalize = Label("finalize")
        CMP(reg_p_len, 0)
        JZ(lfinalize)


        # TODO(dgryski): remove this variable; reuse reg_p_len
        reg_remMod32 = GeneralPurposeRegister64()
        MOV(reg_remMod32, reg_p_len)
        SHL(reg_remMod32, 32)
        ADD(reg_remMod32, reg_p_len)

        reg_xmm0 = XMMRegister()
        MOVQ(reg_xmm0, reg_remMod32)
        PINSRQ(reg_xmm0, reg_remMod32, 1)
        PADDQ(state.v0lo, reg_xmm0)
        PADDQ(state.v0hi, reg_xmm0)

        reg_copylen = GeneralPurposeRegister64()
        MOV(reg_copylen, reg_p_len)
        rotate32By(state.v1lo, reg_copylen)
        MOV(reg_copylen, reg_p_len)
        rotate32By(state.v1hi, reg_copylen)

        # copy(finalPacket[:], bytes[:len(bytes)-remainderMod4])
        MOV(reg_copylen, reg_p_len)
        AND(reg_copylen, 3)
        NEG(reg_copylen)
        ADD(reg_copylen, reg_p_len)

        reg_remainder = GeneralPurposeRegister64()
        MOV(reg_remainder, reg_p)

        PXOR(reg_plo, reg_plo)
        PXOR(reg_phi, reg_phi)
        # reg_p is destroyed
        memcpy32(reg_plo, reg_phi, reg_p, reg_copylen)

        mod4check = Label("mod4check")
        afterMod4 = Label("afterMod4")
        CMP(reg_p_len, 16)
        JL(mod4check)
        # TODO(dgryski): copy(finalPacket[28:], bytes[len(bytes)-4:])
        final = GeneralPurposeRegister32()
        ADD(reg_remainder, reg_p_len)
        SUB(reg_remainder, 4)
        MOV(final, [reg_remainder])
        # load last 4 bytes from remainder
        # shove them into last 4 bytes of final packet
        # == high 4 bytes of
        PINSRD(reg_phi, final, 3)
        JMP(afterMod4)
        LABEL(mod4check)
        MOV(reg_copylen, reg_p_len)
        AND(reg_copylen, 3)
        NEG(reg_copylen)
        ADD(reg_copylen, reg_p_len)
        ADD(reg_remainder, reg_copylen)
        reg_remMod4 = GeneralPurposeRegister64()
        MOV(reg_remMod4, reg_p_len)
        AND(reg_remMod4, 3)
        JZ(afterMod4)
        final = GeneralPurposeRegister64()
        XOR(final, final)
        MOVZX(final, byte[reg_remainder])
        tmp = GeneralPurposeRegister64()
        MOV(tmp, reg_remMod4)
        SHR(tmp, 1)
        offs = GeneralPurposeRegister64()
        MOV(offs, reg_remainder)
        ADD(offs, tmp)
        MOVZX(tmp, byte[offs])
        SHL(tmp, 8)
        OR(final, tmp)
        MOV(tmp, reg_remMod4)
        SUB(tmp, 1)
        MOV(offs, reg_remainder)
        ADD(offs, tmp)
        MOVZX(tmp, byte[offs])
        SHL(tmp, 16)
        OR(final, tmp)
        PINSRQ(reg_phi, final, 0)
        LABEL(afterMod4)

        update(reg_plo, reg_phi, state)

        LABEL(lfinalize)
        ret = finalize(state)
        RETURN(ret)


MakeHash()
