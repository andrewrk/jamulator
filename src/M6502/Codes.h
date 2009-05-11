/** M6502: portable 6502 emulator ****************************/
/**                                                         **/
/**                          Codes.h                        **/
/**                                                         **/
/** This file contains implementation for the main table of **/
/** 6502 commands. It is included from 6502.c.              **/
/**                                                         **/
/** Copyright (C) Marat Fayzullin 1996-2007                 **/
/**               Alex Krasivsky  1996                      **/
/**     You are not allowed to distribute this software     **/
/**     commercially. Please, notify me, if you make any    **/
/**     changes to this file.                               **/
/*************************************************************/

case 0x10: if(R->P&N_FLAG) R->PC.W++; else { M_JR; } break; /* BPL * REL */
case 0x30: if(R->P&N_FLAG) { M_JR; } else R->PC.W++; break; /* BMI * REL */
case 0xD0: if(R->P&Z_FLAG) R->PC.W++; else { M_JR; } break; /* BNE * REL */
case 0xF0: if(R->P&Z_FLAG) { M_JR; } else R->PC.W++; break; /* BEQ * REL */
case 0x90: if(R->P&C_FLAG) R->PC.W++; else { M_JR; } break; /* BCC * REL */
case 0xB0: if(R->P&C_FLAG) { M_JR; } else R->PC.W++; break; /* BCS * REL */
case 0x50: if(R->P&V_FLAG) R->PC.W++; else { M_JR; } break; /* BVC * REL */
case 0x70: if(R->P&V_FLAG) { M_JR; } else R->PC.W++; break; /* BVS * REL */

/* RTI */
case 0x40:
  M_POP(R->P);R->P|=R_FLAG;M_POP(R->PC.B.l);M_POP(R->PC.B.h);
  break;

/* RTS */
case 0x60:
  M_POP(R->PC.B.l);M_POP(R->PC.B.h);R->PC.W++;break;

/* JSR $ssss ABS */
case 0x20:
  K.B.l=Op6502(R->PC.W++);
  K.B.h=Op6502(R->PC.W);
  M_PUSH(R->PC.B.h);
  M_PUSH(R->PC.B.l);
  R->PC=K;break;

/* JMP $ssss ABS */
case 0x4C: M_LDWORD(K);R->PC=K;break;

/* JMP ($ssss) ABDINDIR */
case 0x6C:
  M_LDWORD(K);
  R->PC.B.l=Rd6502(K.W);
  K.B.l++;
  R->PC.B.h=Rd6502(K.W);
  break;

/* BRK */
case 0x00:
  R->PC.W++;
  M_PUSH(R->PC.B.h);M_PUSH(R->PC.B.l);
  M_PUSH(R->P|B_FLAG);
  R->P=(R->P|I_FLAG)&~D_FLAG;
  R->PC.B.l=Rd6502(0xFFFE);
  R->PC.B.h=Rd6502(0xFFFF);
  break;

/* CLI */
case 0x58:
  if((R->IRequest!=INT_NONE)&&(R->P&I_FLAG))
  {
    R->AfterCLI=1;
    R->IBackup=R->ICount;
    R->ICount=1;
  }
  R->P&=~I_FLAG;
  break;

/* PLP */
case 0x28:
  M_POP(I);
  if((R->IRequest!=INT_NONE)&&((I^R->P)&~I&I_FLAG))
  {
    R->AfterCLI=1;
    R->IBackup=R->ICount;
    R->ICount=1;
  }
  R->P=I|R_FLAG|B_FLAG;
  break;

case 0x08: M_PUSH(R->P);break;               /* PHP */
case 0x18: R->P&=~C_FLAG;break;              /* CLC */
case 0xB8: R->P&=~V_FLAG;break;              /* CLV */
case 0xD8: R->P&=~D_FLAG;break;              /* CLD */
case 0x38: R->P|=C_FLAG;break;               /* SEC */
case 0xF8: R->P|=D_FLAG;break;               /* SED */
case 0x78: R->P|=I_FLAG;break;               /* SEI */
case 0x48: M_PUSH(R->A);break;               /* PHA */
case 0x68: M_POP(R->A);M_FL(R->A);break;     /* PLA */
case 0x98: R->A=R->Y;M_FL(R->A);break;       /* TYA */
case 0xA8: R->Y=R->A;M_FL(R->Y);break;       /* TAY */
case 0xC8: R->Y++;M_FL(R->Y);break;          /* INY */
case 0x88: R->Y--;M_FL(R->Y);break;          /* DEY */
case 0x8A: R->A=R->X;M_FL(R->A);break;       /* TXA */
case 0xAA: R->X=R->A;M_FL(R->X);break;       /* TAX */
case 0xE8: R->X++;M_FL(R->X);break;          /* INX */
case 0xCA: R->X--;M_FL(R->X);break;          /* DEX */
case 0xEA: break;                            /* NOP */
case 0x9A: R->S=R->X;break;                  /* TXS */
case 0xBA: R->X=R->S;M_FL(R->X);break;       /* TSX */

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
case 0x84: MW_Zp(R->Y);break;             /* STY $ss ZP */
case 0x85: MW_Zp(R->A);break;             /* STA $ss ZP */
case 0x86: MW_Zp(R->X);break;             /* STX $ss ZP */
case 0xA4: MR_Zp(R->Y);M_FL(R->Y);break;  /* LDY $ss ZP */
case 0xA5: MR_Zp(R->A);M_FL(R->A);break;  /* LDA $ss ZP */
case 0xA6: MR_Zp(R->X);M_FL(R->X);break;  /* LDX $ss ZP */
case 0xC4: MR_Zp(I);M_CMP(R->Y,I);break;  /* CPY $ss ZP */
case 0xC5: MR_Zp(I);M_CMP(R->A,I);break;  /* CMP $ss ZP */
case 0xC6: MM_Zp(M_DEC);break;            /* DEC $ss ZP */
case 0xE4: MR_Zp(I);M_CMP(R->X,I);break;  /* CPX $ss ZP */
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
case 0x8C: MW_Ab(R->Y);break;             /* STY $ssss ABS */
case 0x8D: MW_Ab(R->A);break;             /* STA $ssss ABS */
case 0x8E: MW_Ab(R->X);break;             /* STX $ssss ABS */
case 0xAC: MR_Ab(R->Y);M_FL(R->Y);break;  /* LDY $ssss ABS */
case 0xAD: MR_Ab(R->A);M_FL(R->A);break;  /* LDA $ssss ABS */
case 0xAE: MR_Ab(R->X);M_FL(R->X);break;  /* LDX $ssss ABS */
case 0xCC: MR_Ab(I);M_CMP(R->Y,I);break;  /* CPY $ssss ABS */
case 0xCD: MR_Ab(I);M_CMP(R->A,I);break;  /* CMP $ssss ABS */
case 0xCE: MM_Ab(M_DEC);break;            /* DEC $ssss ABS */
case 0xEC: MR_Ab(I);M_CMP(R->X,I);break;  /* CPX $ssss ABS */
case 0xED: MR_Ab(I);M_SBC(I);break;       /* SBC $ssss ABS */
case 0xEE: MM_Ab(M_INC);break;            /* INC $ssss ABS */

case 0x09: MR_Im(I);M_ORA(I);break;       /* ORA #$ss IMM */
case 0x29: MR_Im(I);M_AND(I);break;       /* AND #$ss IMM */
case 0x49: MR_Im(I);M_EOR(I);break;       /* EOR #$ss IMM */
case 0x69: MR_Im(I);M_ADC(I);break;       /* ADC #$ss IMM */
case 0xA0: MR_Im(R->Y);M_FL(R->Y);break;  /* LDY #$ss IMM */
case 0xA2: MR_Im(R->X);M_FL(R->X);break;  /* LDX #$ss IMM */
case 0xA9: MR_Im(R->A);M_FL(R->A);break;  /* LDA #$ss IMM */
case 0xC0: MR_Im(I);M_CMP(R->Y,I);break;  /* CPY #$ss IMM */
case 0xC9: MR_Im(I);M_CMP(R->A,I);break;  /* CMP #$ss IMM */
case 0xE0: MR_Im(I);M_CMP(R->X,I);break;  /* CPX #$ss IMM */
case 0xE9: MR_Im(I);M_SBC(I);break;       /* SBC #$ss IMM */

case 0x15: MR_Zx(I);M_ORA(I);break;       /* ORA $ss,x ZP,x */
case 0x16: MM_Zx(M_ASL);break;            /* ASL $ss,x ZP,x */
case 0x35: MR_Zx(I);M_AND(I);break;       /* AND $ss,x ZP,x */
case 0x36: MM_Zx(M_ROL);break;            /* ROL $ss,x ZP,x */
case 0x55: MR_Zx(I);M_EOR(I);break;       /* EOR $ss,x ZP,x */
case 0x56: MM_Zx(M_LSR);break;            /* LSR $ss,x ZP,x */
case 0x75: MR_Zx(I);M_ADC(I);break;       /* ADC $ss,x ZP,x */
case 0x76: MM_Zx(M_ROR);break;            /* ROR $ss,x ZP,x */
case 0x94: MW_Zx(R->Y);break;             /* STY $ss,x ZP,x */
case 0x95: MW_Zx(R->A);break;             /* STA $ss,x ZP,x */
case 0x96: MW_Zy(R->X);break;             /* STX $ss,y ZP,y */
case 0xB4: MR_Zx(R->Y);M_FL(R->Y);break;  /* LDY $ss,x ZP,x */
case 0xB5: MR_Zx(R->A);M_FL(R->A);break;  /* LDA $ss,x ZP,x */
case 0xB6: MR_Zy(R->X);M_FL(R->X);break;  /* LDX $ss,y ZP,y */
case 0xD5: MR_Zx(I);M_CMP(R->A,I);break;  /* CMP $ss,x ZP,x */
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
case 0x99: MW_Ay(R->A);break;             /* STA $ssss,y ABS,y */
case 0x9D: MW_Ax(R->A);break;             /* STA $ssss,x ABS,x */
case 0xB9: MR_Ay(R->A);M_FL(R->A);break;  /* LDA $ssss,y ABS,y */
case 0xBC: MR_Ax(R->Y);M_FL(R->Y);break;  /* LDY $ssss,x ABS,x */
case 0xBD: MR_Ax(R->A);M_FL(R->A);break;  /* LDA $ssss,x ABS,x */
case 0xBE: MR_Ay(R->X);M_FL(R->X);break;  /* LDX $ssss,y ABS,y */
case 0xD9: MR_Ay(I);M_CMP(R->A,I);break;  /* CMP $ssss,y ABS,y */
case 0xDD: MR_Ax(I);M_CMP(R->A,I);break;  /* CMP $ssss,x ABS,x */
case 0xDE: MM_Ax(M_DEC);break;            /* DEC $ssss,x ABS,x */
case 0xF9: MR_Ay(I);M_SBC(I);break;       /* SBC $ssss,y ABS,y */
case 0xFD: MR_Ax(I);M_SBC(I);break;       /* SBC $ssss,x ABS,x */
case 0xFE: MM_Ax(M_INC);break;            /* INC $ssss,x ABS,x */

case 0x01: MR_Ix(I);M_ORA(I);break;       /* ORA ($ss,x) INDEXINDIR */
case 0x11: MR_Iy(I);M_ORA(I);break;       /* ORA ($ss),y INDIRINDEX */
case 0x21: MR_Ix(I);M_AND(I);break;       /* AND ($ss,x) INDEXINDIR */
case 0x31: MR_Iy(I);M_AND(I);break;       /* AND ($ss),y INDIRINDEX */
case 0x41: MR_Ix(I);M_EOR(I);break;       /* EOR ($ss,x) INDEXINDIR */
case 0x51: MR_Iy(I);M_EOR(I);break;       /* EOR ($ss),y INDIRINDEX */
case 0x61: MR_Ix(I);M_ADC(I);break;       /* ADC ($ss,x) INDEXINDIR */
case 0x71: MR_Iy(I);M_ADC(I);break;       /* ADC ($ss),y INDIRINDEX */
case 0x81: MW_Ix(R->A);break;             /* STA ($ss,x) INDEXINDIR */
case 0x91: MW_Iy(R->A);break;             /* STA ($ss),y INDIRINDEX */
case 0xA1: MR_Ix(R->A);M_FL(R->A);break;  /* LDA ($ss,x) INDEXINDIR */
case 0xB1: MR_Iy(R->A);M_FL(R->A);break;  /* LDA ($ss),y INDIRINDEX */
case 0xC1: MR_Ix(I);M_CMP(R->A,I);break;  /* CMP ($ss,x) INDEXINDIR */
case 0xD1: MR_Iy(I);M_CMP(R->A,I);break;  /* CMP ($ss),y INDIRINDEX */
case 0xE1: MR_Ix(I);M_SBC(I);break;       /* SBC ($ss,x) INDEXINDIR */
case 0xF1: MR_Iy(I);M_SBC(I);break;       /* SBC ($ss),y INDIRINDEX */

case 0x0A: M_ASL(R->A);break;             /* ASL a ACC */
case 0x2A: M_ROL(R->A);break;             /* ROL a ACC */
case 0x4A: M_LSR(R->A);break;             /* LSR a ACC */
case 0x6A: M_ROR(R->A);break;             /* ROR a ACC */

default:
  /* Try to execute a patch function. If it fails, treat */
  /* the opcode as undefined.                            */
  if(!Patch6502(Op6502(R->PC.W-1),R))
    if(R->TrapBadOps)
      printf
      (
        "[M6502 %lX] Unrecognized instruction: $%02X at PC=$%04X\n",
        (unsigned long)(R->User),Op6502(R->PC.W-1),(word)(R->PC.W-1)
      );
  break;
