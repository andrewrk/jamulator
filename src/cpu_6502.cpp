#include <iostream>
using namespace std;

#include "cpu_6502.h"

/** Addressing Methods ***************************************/
/** These macros calculate and return effective addresses.  **/
/*************************************************************/
#define MC_Ab(Rg)	M_LDWORD(Rg)
#define MC_Zp(Rg)       Rg.w=readFunc(pc.w++)
#define MC_Zx(Rg)       Rg.w=(byte)(readFunc(pc.w++)+xr)
#define MC_Zy(Rg)       Rg.w=(byte)(readFunc(pc.w++)+yr)
#define MC_Ax(Rg)	M_LDWORD(Rg);Rg.w+=xr
#define MC_Ay(Rg)	M_LDWORD(Rg);Rg.w+=yr
#define MC_Ix(Rg)       K.w=(byte)(readFunc(pc.w++)+xr); \
			Rg.b.l=readFunc(K.w++);Rg.b.h=readFunc(K.w)
#define MC_Iy(Rg)       K.w=readFunc(pc.w++); \
			Rg.b.l=readFunc(K.w++);Rg.b.h=readFunc(K.w); \
			Rg.w+=yr

/** Reading From Memory **************************************/
/** These macros calculate address and read from it.        **/
/*************************************************************/
#define MR_Ab(Rg)	MC_Ab(J);Rg=readFunc(J.w)
#define MR_Im(Rg)	Rg=readFunc(pc.w++)
#define	MR_Zp(Rg)	MC_Zp(J);Rg=readFunc(J.w)
#define MR_Zx(Rg)	MC_Zx(J);Rg=readFunc(J.w)
#define MR_Zy(Rg)	MC_Zy(J);Rg=readFunc(J.w)
#define	MR_Ax(Rg)	MC_Ax(J);Rg=readFunc(J.w)
#define MR_Ay(Rg)	MC_Ay(J);Rg=readFunc(J.w)
#define MR_Ix(Rg)	MC_Ix(J);Rg=readFunc(J.w)
#define MR_Iy(Rg)	MC_Iy(J);Rg=readFunc(J.w)

/** Writing To Memory ****************************************/
/** These macros calculate address and write to it.         **/
/*************************************************************/
#define MW_Ab(Rg)	MC_Ab(J);writeFunc(J.w,Rg)
#define MW_Zp(Rg)	MC_Zp(J);writeFunc(J.w,Rg)
#define MW_Zx(Rg)	MC_Zx(J);writeFunc(J.w,Rg)
#define MW_Zy(Rg)	MC_Zy(J);writeFunc(J.w,Rg)
#define MW_Ax(Rg)	MC_Ax(J);writeFunc(J.w,Rg)
#define MW_Ay(Rg)	MC_Ay(J);writeFunc(J.w,Rg)
#define MW_Ix(Rg)	MC_Ix(J);writeFunc(J.w,Rg)
#define MW_Iy(Rg)	MC_Iy(J);writeFunc(J.w,Rg)

/** Modifying Memory *****************************************/
/** These macros calculate address and modify it.           **/
/*************************************************************/
#define MM_Ab(Cmd)	MC_Ab(J);I=readFunc(J.w);Cmd(I);writeFunc(J.w,I)
#define MM_Zp(Cmd)	MC_Zp(J);I=readFunc(J.w);Cmd(I);writeFunc(J.w,I)
#define MM_Zx(Cmd)	MC_Zx(J);I=readFunc(J.w);Cmd(I);writeFunc(J.w,I)
#define MM_Ax(Cmd)	MC_Ax(J);I=readFunc(J.w);Cmd(I);writeFunc(J.w,I)

/** Other Macros *********************************************/
/** Calculating flags, stack, jumps, arithmetics, etc.      **/
/*************************************************************/
#define M_FL(Rg)	st=(st&~(FlagZ|FlagN))|zn_table[Rg]
#define M_LDWORD(Rg)	Rg.b.l=readFunc(pc.w++);Rg.b.h=readFunc(pc.w++)

#define M_PUSH(Rg)	writeFunc( 0x0100|sp,Rg);sp--
#define M_POP(Rg)	sp++;Rg=readFunc( 0x0100|sp)
#define M_JR		pc.w+=(offset)readFunc( pc.w)+1;cycles_left--

#ifdef NO_DECIMAL

#define M_ADC(Rg) \
  K.w=ac+Rg+(st&FlagC); \
  st&=~(FlagN|FlagV|FlagZ|FlagC); \
  st|=(~(ac^Rg)&(ac^K.b.l)&0x80? FlagV:0)| \
        (K.b.h? FlagC:0)|zn_table[K.b.l]; \
  ac=K.b.l

/* Warning! FlagC is inverted before SBC and after it */
#define M_SBC(Rg) \
  K.w=ac-Rg-(~st&FlagC); \
  st&=~(FlagN|FlagV|FlagZ|FlagC); \
  st|=((ac^Rg)&(ac^K.b.l)&0x80? FlagV:0)| \
        (K.b.h? 0:FlagC)|zn_table[K.b.l]; \
  ac=K.b.l

#else /* NO_DECIMAL */

#define M_ADC(Rg) \
  if(st&FlagD) \
  { \
    K.b.l=(ac&0x0F)+(Rg&0x0F)+(st&FlagC); \
    if(K.b.l>9) K.b.l+=6; \
    K.b.h=(ac>>4)+(Rg>>4)+(K.b.l>15? 1:0); \
    ac=(K.b.l&0x0F)|(K.b.h<<4); \
    st=(st&~FlagC)|(K.b.h>15? FlagC:0); \
  } \
  else \
  { \
    K.w=ac+Rg+(st&FlagC); \
    st&=~(FlagN|FlagV|FlagZ|FlagC); \
    st|=(~(ac^Rg)&(ac^K.b.l)&0x80? FlagV:0)| \
          (K.b.h? FlagC:0)|zn_table[K.b.l]; \
    ac=K.b.l; \
  }

/* Warning! FlagC is inverted before SBC and after it */
#define M_SBC(Rg) \
  if(st&FlagD) \
  { \
    K.b.l=(ac&0x0F)-(Rg&0x0F)-(~st&FlagC); \
    if(K.b.l&0x10) K.b.l-=6; \
    K.b.h=(ac>>4)-(Rg>>4)-((K.b.l&0x10)>>4); \
    if(K.b.h&0x10) K.b.h-=6; \
    ac=(K.b.l&0x0F)|(K.b.h<<4); \
    st=(st&~FlagC)|(K.b.h>15? 0:FlagC); \
  } \
  else \
  { \
    K.w=ac-Rg-(~st&FlagC); \
    st&=~(FlagN|FlagV|FlagZ|FlagC); \
    st|=((ac^Rg)&(ac^K.b.l)&0x80? FlagV:0)| \
          (K.b.h? 0:FlagC)|zn_table[K.b.l]; \
    ac=K.b.l; \
  }

#endif /* NO_DECIMAL */

#define M_CMP(Rg1,Rg2) \
  K.w=Rg1-Rg2; \
  st&=~(FlagN|FlagZ|FlagC); \
  st|=zn_table[K.b.l]|(K.b.h? 0:FlagC)
#define M_BIT(Rg) \
  st&=~(FlagN|FlagV|FlagZ); \
  st|=(Rg&(FlagN|FlagV))|(Rg&ac? 0:FlagZ)

#define M_AND(Rg)	ac&=Rg;M_FL(ac)
#define M_ORA(Rg)	ac|=Rg;M_FL(ac)
#define M_EOR(Rg)	ac^=Rg;M_FL(ac)
#define M_INC(Rg)	Rg++;M_FL(Rg)
#define M_DEC(Rg)	Rg--;M_FL(Rg)

#define M_ASL(Rg)	st&=~FlagC;st|=Rg>>7;Rg<<=1;M_FL(Rg)
#define M_LSR(Rg)	st&=~FlagC;st|=Rg&FlagC;Rg>>=1;M_FL(Rg)
#define M_ROL(Rg)	K.b.l=(Rg<<1)|(st&FlagC); \
			st&=~FlagC;st|=Rg>>7;Rg=K.b.l; \
			M_FL(Rg)
#define M_ROR(Rg)	K.b.l=(Rg>>1)|(st<<7); \
			st&=~FlagC;st|=Rg&FlagC;Rg=K.b.l; \
			M_FL(Rg)

Cpu6502::Cpu6502(
		int _clock_speed,
		int _interupt_check, 
		InteruptType (*_loopCallback)(Cpu6502* cpu),
		byte (*_readFunc)(Cpu6502 * cpu, word address),
		void (*_writeFunc)(Cpu6502 * cpu, word address, byte value) ) :
	clock_speed(_clock_speed),
	interupt_check(_interupt_check),
	loopCallback(_loopCallback),
	readFunc(_readFunc),
	writeFunc(_writeFunc)
{
	reset();
}

Cpu6502::~Cpu6502(){
	// nothing to do
}


void Cpu6502::reset() {
	ac = xr = yr = 0x00;
	st = FlagZ|FlagR;
	sp = 0xFF;
	pc.b.l = readFunc( 0xFFFC);
	pc.b.h = readFunc( 0xFFFD);
	
	cycles_left = interupt_period;
}

void Cpu6502::interupt(byte type){
	register pair J;

	if( type == IntNmi || ( type == IntIrq && !(st&FlagI) ) ) {
		cycles_left -= 7;
		M_PUSH(pc.b.h);
		M_PUSH(pc.b.l);
		M_PUSH(st & ~FlagB);
		st &= ~FlagD;

		if( type == IntNmi ) {
			J.w = 0xFFFA; 
		} else { 
			st |= FlagI;
			J.w = 0xFFFE;
		}
		pc.b.l=readFunc( J.w++);
		pc.b.h=readFunc( J.w);
	}
}

int Cpu6502::run() {
	register pair J,K;
	register byte I;

	for(;;)	{
		I = readFunc( pc.w++);
		cycles_left -= op_cycles[I];
		switch(I) {
			case 0x10: /* BPL * REL */
				if(st&FlagN) pc.w++; else { M_JR; } break; 
			case 0x30: /* BMI * REL */
				if(st&FlagN) { M_JR; } else pc.w++; break; 
			case 0xD0: /* BNE * REL */
				if(st&FlagZ) pc.w++; else { M_JR; } break; 
			case 0xF0:/* BEQ * REL */
				if(st&FlagZ) { M_JR; } else pc.w++; break; 
			case 0x90:/* BCC * REL */
				if(st&FlagC) pc.w++; else { M_JR; } break; 
			case 0xB0:/* BCS * REL */
				if(st&FlagC) { M_JR; } else pc.w++; break; 
			case 0x50:/* BVC * REL */
				if(st&FlagV) pc.w++; else { M_JR; } break; 
			case 0x70:/* BVS * REL */
				if(st&FlagV) { M_JR; } else pc.w++; break; 

						  
			case 0x40: /* RTI */
						   M_POP(st);st|=FlagR;M_POP(pc.b.l);M_POP(pc.b.h);
						   break;

						  
			case 0x60: /* RTS */
						   M_POP(pc.b.l);M_POP(pc.b.h);pc.w++;break;

						   
			case 0x20:/* JSR $ssss ABS */
						   K.b.l=readFunc(pc.w++);
						   K.b.h=readFunc(pc.w);
						   M_PUSH(pc.b.h);
						   M_PUSH(pc.b.l);
						   pc=K;break;

						 
			case 0x4C:  /* JMP $ssss ABS */
			M_LDWORD(K);pc=K;break;

					  
			case 0x6C: /* JMP ($ssss) ABDINDIR */
					   M_LDWORD(K);
					   pc.b.l=readFunc(K.w);
					   K.b.l++;
					   pc.b.h=readFunc(K.w);
					   break;

					  
			case 0x00: /* BRK */
					   pc.w++;
					   M_PUSH(pc.b.h);M_PUSH(pc.b.l);
					   M_PUSH(st|FlagB);
					   st=(st|FlagI)&~FlagD;
					   pc.b.l=readFunc(0xFFFE);
					   pc.b.h=readFunc(0xFFFF);
					   break;

					  
			case 0x58 /* CLI */:
					   st&=~FlagI;
					   break;

					  
			case 0x28: /* PLP */
					   M_POP(I);
					   st=I|FlagR|FlagB;
					   break;

			case 0x08: M_PUSH(st);break;               /* PHP */
			case 0x18: st&=~FlagC;break;              /* CLC */
			case 0xB8: st&=~FlagV;break;              /* CLV */
			case 0xD8: st&=~FlagD;break;              /* CLD */
			case 0x38: st|=FlagC;break;               /* SEC */
			case 0xF8: st|=FlagD;break;               /* SED */
			case 0x78: st|=FlagI;break;               /* SEI */
			case 0x48: M_PUSH(ac);break;               /* PHA */
			case 0x68: M_POP(ac);M_FL(ac);break;     /* PLA */
			case 0x98: ac=yr;M_FL(ac);break;       /* TYA */
			case 0xA8: yr=ac;M_FL(yr);break;       /* TAY */
			case 0xC8: yr++;M_FL(yr);break;          /* INY */
			case 0x88: yr--;M_FL(yr);break;          /* DEY */
			case 0x8A: ac=xr;M_FL(ac);break;       /* TXA */
			case 0xAA: xr=ac;M_FL(xr);break;       /* TAX */
			case 0xE8: xr++;M_FL(xr);break;          /* INX */
			case 0xCA: xr--;M_FL(xr);break;          /* DEX */
			case 0xEA: break;                            /* NOP */
			case 0x9A: sp=xr;break;                  /* TXS */
			case 0xBA: xr=sp;M_FL(xr);break;       /* TSX */

			case 0x24: MR_Zp(I);M_BIT(I);break;       /* BIT $ss ZP */
			case 0x2C: MR_Ab(I);M_BIT(I);break;       /* BIT $ssss ABS */

			case 0x05: MR_Zp(I);M_ORA(I);break;       /* ORA $ss ZP */
			case 0x06: MM_Zp(M_ASL);break;            /* ASL $ss ZP */
			case 0x25: MR_Zp(I);M_AND(I);break;       /* AND $ss ZP */
			case 0x26: MM_Zp(M_ROL);break;            /* ROL $ss ZP */
			case 0x45: MR_Zp(I);M_EOR(I);break;       /* EOR $ss ZP */
			case 0x46: MM_Zp(M_LSR);break;            /* LSR $ss ZP */
			case 0x65: MR_Zp(I);M_ADC(I);break;       /* ADC $ss ZP */
			case 0x66: MM_Zp(M_ROR);break;            /* ROR $ss ZP */
			case 0x84: MW_Zp(yr);break;             /* STY $ss ZP */
			case 0x85: MW_Zp(ac);break;             /* STA $ss ZP */
			case 0x86: MW_Zp(xr);break;             /* STX $ss ZP */
			case 0xA4: MR_Zp(yr);M_FL(yr);break;  /* LDY $ss ZP */
			case 0xA5: MR_Zp(ac);M_FL(ac);break;  /* LDA $ss ZP */
			case 0xA6: MR_Zp(xr);M_FL(xr);break;  /* LDX $ss ZP */
			case 0xC4: MR_Zp(I);M_CMP(yr,I);break;  /* CPY $ss ZP */
			case 0xC5: MR_Zp(I);M_CMP(ac,I);break;  /* CMP $ss ZP */
			case 0xC6: MM_Zp(M_DEC);break;            /* DEC $ss ZP */
			case 0xE4: MR_Zp(I);M_CMP(xr,I);break;  /* CPX $ss ZP */
			case 0xE5: MR_Zp(I);M_SBC(I);break;       /* SBC $ss ZP */
			case 0xE6: MM_Zp(M_INC);break;            /* INC $ss ZP */

			case 0x0D: MR_Ab(I);M_ORA(I);break;       /* ORA $ssss ABS */
			case 0x0E: MM_Ab(M_ASL);break;            /* ASL $ssss ABS */
			case 0x2D: MR_Ab(I);M_AND(I);break;       /* AND $ssss ABS */
			case 0x2E: MM_Ab(M_ROL);break;            /* ROL $ssss ABS */
			case 0x4D: MR_Ab(I);M_EOR(I);break;       /* EOR $ssss ABS */
			case 0x4E: MM_Ab(M_LSR);break;            /* LSR $ssss ABS */
			case 0x6D: MR_Ab(I);M_ADC(I);break;       /* ADC $ssss ABS */
			case 0x6E: MM_Ab(M_ROR);break;            /* ROR $ssss ABS */
			case 0x8C: MW_Ab(yr);break;             /* STY $ssss ABS */
			case 0x8D: MW_Ab(ac);break;             /* STA $ssss ABS */
			case 0x8E: MW_Ab(xr);break;             /* STX $ssss ABS */
			case 0xAC: MR_Ab(yr);M_FL(yr);break;  /* LDY $ssss ABS */
			case 0xAD: MR_Ab(ac);M_FL(ac);break;  /* LDA $ssss ABS */
			case 0xAE: MR_Ab(xr);M_FL(xr);break;  /* LDX $ssss ABS */
			case 0xCC: MR_Ab(I);M_CMP(yr,I);break;  /* CPY $ssss ABS */
			case 0xCD: MR_Ab(I);M_CMP(ac,I);break;  /* CMP $ssss ABS */
			case 0xCE: MM_Ab(M_DEC);break;            /* DEC $ssss ABS */
			case 0xEC: MR_Ab(I);M_CMP(xr,I);break;  /* CPX $ssss ABS */
			case 0xED: MR_Ab(I);M_SBC(I);break;       /* SBC $ssss ABS */
			case 0xEE: MM_Ab(M_INC);break;            /* INC $ssss ABS */

			case 0x09: MR_Im(I);M_ORA(I);break;       /* ORA #$ss IMM */
			case 0x29: MR_Im(I);M_AND(I);break;       /* AND #$ss IMM */
			case 0x49: MR_Im(I);M_EOR(I);break;       /* EOR #$ss IMM */
			case 0x69: MR_Im(I);M_ADC(I);break;       /* ADC #$ss IMM */
			case 0xA0: MR_Im(yr);M_FL(yr);break;  /* LDY #$ss IMM */
			case 0xA2: MR_Im(xr);M_FL(xr);break;  /* LDX #$ss IMM */
			case 0xA9: MR_Im(ac);M_FL(ac);break;  /* LDA #$ss IMM */
			case 0xC0: MR_Im(I);M_CMP(yr,I);break;  /* CPY #$ss IMM */
			case 0xC9: MR_Im(I);M_CMP(ac,I);break;  /* CMP #$ss IMM */
			case 0xE0: MR_Im(I);M_CMP(xr,I);break;  /* CPX #$ss IMM */
			case 0xE9: MR_Im(I);M_SBC(I);break;       /* SBC #$ss IMM */

			case 0x15: MR_Zx(I);M_ORA(I);break;       /* ORA $ss,x ZP,x */
			case 0x16: MM_Zx(M_ASL);break;            /* ASL $ss,x ZP,x */
			case 0x35: MR_Zx(I);M_AND(I);break;       /* AND $ss,x ZP,x */
			case 0x36: MM_Zx(M_ROL);break;            /* ROL $ss,x ZP,x */
			case 0x55: MR_Zx(I);M_EOR(I);break;       /* EOR $ss,x ZP,x */
			case 0x56: MM_Zx(M_LSR);break;            /* LSR $ss,x ZP,x */
			case 0x75: MR_Zx(I);M_ADC(I);break;       /* ADC $ss,x ZP,x */
			case 0x76: MM_Zx(M_ROR);break;            /* ROR $ss,x ZP,x */
			case 0x94: MW_Zx(yr);break;             /* STY $ss,x ZP,x */
			case 0x95: MW_Zx(ac);break;             /* STA $ss,x ZP,x */
			case 0x96: MW_Zy(xr);break;             /* STX $ss,y ZP,y */
			case 0xB4: MR_Zx(yr);M_FL(yr);break;  /* LDY $ss,x ZP,x */
			case 0xB5: MR_Zx(ac);M_FL(ac);break;  /* LDA $ss,x ZP,x */
			case 0xB6: MR_Zy(xr);M_FL(xr);break;  /* LDX $ss,y ZP,y */
			case 0xD5: MR_Zx(I);M_CMP(ac,I);break;  /* CMP $ss,x ZP,x */
			case 0xD6: MM_Zx(M_DEC);break;            /* DEC $ss,x ZP,x */
			case 0xF5: MR_Zx(I);M_SBC(I);break;       /* SBC $ss,x ZP,x */
			case 0xF6: MM_Zx(M_INC);break;            /* INC $ss,x ZP,x */

			case 0x19: MR_Ay(I);M_ORA(I);break;       /* ORA $ssss,y ABS,y */
			case 0x1D: MR_Ax(I);M_ORA(I);break;       /* ORA $ssss,x ABS,x */
			case 0x1E: MM_Ax(M_ASL);break;            /* ASL $ssss,x ABS,x */
			case 0x39: MR_Ay(I);M_AND(I);break;       /* AND $ssss,y ABS,y */
			case 0x3D: MR_Ax(I);M_AND(I);break;       /* AND $ssss,x ABS,x */
			case 0x3E: MM_Ax(M_ROL);break;            /* ROL $ssss,x ABS,x */
			case 0x59: MR_Ay(I);M_EOR(I);break;       /* EOR $ssss,y ABS,y */
			case 0x5D: MR_Ax(I);M_EOR(I);break;       /* EOR $ssss,x ABS,x */
			case 0x5E: MM_Ax(M_LSR);break;            /* LSR $ssss,x ABS,x */
			case 0x79: MR_Ay(I);M_ADC(I);break;       /* ADC $ssss,y ABS,y */
			case 0x7D: MR_Ax(I);M_ADC(I);break;       /* ADC $ssss,x ABS,x */
			case 0x7E: MM_Ax(M_ROR);break;            /* ROR $ssss,x ABS,x */
			case 0x99: MW_Ay(ac);break;             /* STA $ssss,y ABS,y */
			case 0x9D: MW_Ax(ac);break;             /* STA $ssss,x ABS,x */
			case 0xB9: MR_Ay(ac);M_FL(ac);break;  /* LDA $ssss,y ABS,y */
			case 0xBC: MR_Ax(yr);M_FL(yr);break;  /* LDY $ssss,x ABS,x */
			case 0xBD: MR_Ax(ac);M_FL(ac);break;  /* LDA $ssss,x ABS,x */
			case 0xBE: MR_Ay(xr);M_FL(xr);break;  /* LDX $ssss,y ABS,y */
			case 0xD9: MR_Ay(I);M_CMP(ac,I);break;  /* CMP $ssss,y ABS,y */
			case 0xDD: MR_Ax(I);M_CMP(ac,I);break;  /* CMP $ssss,x ABS,x */
			case 0xDE: MM_Ax(M_DEC);break;            /* DEC $ssss,x ABS,x */
			case 0xF9: MR_Ay(I);M_SBC(I);break;       /* SBC $ssss,y ABS,y */
			case 0xFD: MR_Ax(I);M_SBC(I);break;       /* SBC $ssss,x ABS,x */
			case 0xFE: MM_Ax(M_INC);break;            /* INC $ssss,x ABS,x */

			case 0x01: MR_Ix(I);M_ORA(I);break;    /* ORA ($ss,x) INDEXINDIR */
			case 0x11: MR_Iy(I);M_ORA(I);break;    /* ORA ($ss),y INDIRINDEX */
			case 0x21: MR_Ix(I);M_AND(I);break;    /* AND ($ss,x) INDEXINDIR */
			case 0x31: MR_Iy(I);M_AND(I);break;    /* AND ($ss),y INDIRINDEX */
			case 0x41: MR_Ix(I);M_EOR(I);break;    /* EOR ($ss,x) INDEXINDIR */
			case 0x51: MR_Iy(I);M_EOR(I);break;    /* EOR ($ss),y INDIRINDEX */
			case 0x61: MR_Ix(I);M_ADC(I);break;    /* ADC ($ss,x) INDEXINDIR */
			case 0x71: MR_Iy(I);M_ADC(I);break;    /* ADC ($ss),y INDIRINDEX */
			case 0x81: MW_Ix(ac);break;          /* STA ($ss,x) INDEXINDIR */
			case 0x91: MW_Iy(ac);break;          /* STA ($ss),y INDIRINDEX */

			/* LDA ($ss,x) INDEXINDIR */
			case 0xA1: MR_Ix(ac);M_FL(ac);break;
			/* LDA ($ss),y INDIRINDEX */
			case 0xB1: MR_Iy(ac);M_FL(ac);break;  
			/* CMP ($ss,x) INDEXINDIR */
			case 0xC1: MR_Ix(I);M_CMP(ac,I);break;  
			/* CMP ($ss),y INDIRINDEX */
			case 0xD1: MR_Iy(I);M_CMP(ac,I);break;  
			/* SBC ($ss,x) INDEXINDIR */
			case 0xE1: MR_Ix(I);M_SBC(I);break;       
			/* SBC ($ss),y INDIRINDEX */
			case 0xF1: MR_Iy(I);M_SBC(I);break;       

			case 0x0A: M_ASL(ac);break;             /* ASL a ACC */
			case 0x2A: M_ROL(ac);break;             /* ROL a ACC */
			case 0x4A: M_LSR(ac);break;             /* LSR a ACC */
			case 0x6A: M_ROR(ac);break;             /* ROR a ACC */

			default:
			   // error
			   cerr << "Unrecognized instruction: " << inst 
			   		<< " at PC=" << pc.w-1 << endl;
			   break;
		}

		// check for an interupt
		if( cycles_left <= 0 ) {
			// see if there is an interupt
			I = loopCallback(this);
			// reset the cycle counter
			cycles_left += interupt_period;

			// return from function if IntQuit
			if( I == IntQuit )
				return cycles_left;

			// perform the interupt
			if( I ) 
				interupt(int_type);
		}
	}

	return cycles_left;
}

