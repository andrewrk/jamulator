/** M6502: portable 6502 emulator ****************************/
/**                                                         **/
/**                         M6502.c                         **/
/**                                                         **/
/** This file contains implementation for 6502 CPU. Don't   **/
/** forget to provide Rd6502(), Wr6502(), Loop6502(), and   **/
/** possibly Op6502() functions to accomodate the emulated  **/
/** machine's architecture.                                 **/
/**                                                         **/
/** Copyright (C) Marat Fayzullin 1996-2007                 **/
/**               Alex Krasivsky  1996                      **/
/**     You are not allowed to distribute this software     **/
/**     commercially. Please, notify me, if you make any    **/   
/**     changes to this file.                               **/
/*************************************************************/

#include "M6502.h"
#include "Tables.h"
#include <stdio.h>

/** INLINE ***************************************************/
/** C99 standard has "inline", but older compilers used     **/
/** __inline for the same purpose.                          **/
/*************************************************************/
#ifdef __C99__
#define INLINE static inline
#else
#define INLINE static __inline
#endif

/** System-Dependent Stuff ***********************************/
/** This is system-dependent code put here to speed things  **/
/** up. It has to stay inlined to be fast.                  **/
/*************************************************************/
#ifdef INES
#define FAST_RDOP
extern byte *Page[];
INLINE byte Op6502(register word A)
{
  return(Page[A>>13][A&0x1FFF]);
}
#endif /* INES */

#ifdef S60
extern int Opt6502(M6502 *R);
#endif

/** FAST_RDOP ************************************************/
/** With this #define not present, Rd6502() should perform  **/
/** the functions of Op6502().                              **/
/*************************************************************/
#ifndef FAST_RDOP
#define Op6502(A) Rd6502(A)
#endif

/** Addressing Methods ***************************************/
/** These macros calculate and return effective addresses.  **/
/*************************************************************/
#define MC_Ab(Rg)	M_LDWORD(Rg)
#define MC_Zp(Rg)       Rg.W=Op6502(R->PC.W++)
#define MC_Zx(Rg)       Rg.W=(byte)(Op6502(R->PC.W++)+R->X)
#define MC_Zy(Rg)       Rg.W=(byte)(Op6502(R->PC.W++)+R->Y)
#define MC_Ax(Rg)	M_LDWORD(Rg);Rg.W+=R->X
#define MC_Ay(Rg)	M_LDWORD(Rg);Rg.W+=R->Y
#define MC_Ix(Rg)       K.W=(byte)(Op6502(R->PC.W++)+R->X); \
			Rg.B.l=Op6502(K.W++);Rg.B.h=Op6502(K.W)
#define MC_Iy(Rg)       K.W=Op6502(R->PC.W++); \
			Rg.B.l=Op6502(K.W++);Rg.B.h=Op6502(K.W); \
			Rg.W+=R->Y

/** Reading From Memory **************************************/
/** These macros calculate address and read from it.        **/
/*************************************************************/
#define MR_Ab(Rg)	MC_Ab(J);Rg=Rd6502(J.W)
#define MR_Im(Rg)	Rg=Op6502(R->PC.W++)
#define	MR_Zp(Rg)	MC_Zp(J);Rg=Rd6502(J.W)
#define MR_Zx(Rg)	MC_Zx(J);Rg=Rd6502(J.W)
#define MR_Zy(Rg)	MC_Zy(J);Rg=Rd6502(J.W)
#define	MR_Ax(Rg)	MC_Ax(J);Rg=Rd6502(J.W)
#define MR_Ay(Rg)	MC_Ay(J);Rg=Rd6502(J.W)
#define MR_Ix(Rg)	MC_Ix(J);Rg=Rd6502(J.W)
#define MR_Iy(Rg)	MC_Iy(J);Rg=Rd6502(J.W)

/** Writing To Memory ****************************************/
/** These macros calculate address and write to it.         **/
/*************************************************************/
#define MW_Ab(Rg)	MC_Ab(J);Wr6502(J.W,Rg)
#define MW_Zp(Rg)	MC_Zp(J);Wr6502(J.W,Rg)
#define MW_Zx(Rg)	MC_Zx(J);Wr6502(J.W,Rg)
#define MW_Zy(Rg)	MC_Zy(J);Wr6502(J.W,Rg)
#define MW_Ax(Rg)	MC_Ax(J);Wr6502(J.W,Rg)
#define MW_Ay(Rg)	MC_Ay(J);Wr6502(J.W,Rg)
#define MW_Ix(Rg)	MC_Ix(J);Wr6502(J.W,Rg)
#define MW_Iy(Rg)	MC_Iy(J);Wr6502(J.W,Rg)

/** Modifying Memory *****************************************/
/** These macros calculate address and modify it.           **/
/*************************************************************/
#define MM_Ab(Cmd)	MC_Ab(J);I=Rd6502(J.W);Cmd(I);Wr6502(J.W,I)
#define MM_Zp(Cmd)	MC_Zp(J);I=Rd6502(J.W);Cmd(I);Wr6502(J.W,I)
#define MM_Zx(Cmd)	MC_Zx(J);I=Rd6502(J.W);Cmd(I);Wr6502(J.W,I)
#define MM_Ax(Cmd)	MC_Ax(J);I=Rd6502(J.W);Cmd(I);Wr6502(J.W,I)

/** Other Macros *********************************************/
/** Calculating flags, stack, jumps, arithmetics, etc.      **/
/*************************************************************/
#define M_FL(Rg)	R->P=(R->P&~(Z_FLAG|N_FLAG))|ZNTable[Rg]
#define M_LDWORD(Rg)	Rg.B.l=Op6502(R->PC.W++);Rg.B.h=Op6502(R->PC.W++)

#define M_PUSH(Rg)	Wr6502(0x0100|R->S,Rg);R->S--
#define M_POP(Rg)	R->S++;Rg=Op6502(0x0100|R->S)
#define M_JR		R->PC.W+=(offset)Op6502(R->PC.W)+1;R->ICount--

#ifdef NO_DECIMAL

#define M_ADC(Rg) \
  K.W=R->A+Rg+(R->P&C_FLAG); \
  R->P&=~(N_FLAG|V_FLAG|Z_FLAG|C_FLAG); \
  R->P|=(~(R->A^Rg)&(R->A^K.B.l)&0x80? V_FLAG:0)| \
        (K.B.h? C_FLAG:0)|ZNTable[K.B.l]; \
  R->A=K.B.l

/* Warning! C_FLAG is inverted before SBC and after it */
#define M_SBC(Rg) \
  K.W=R->A-Rg-(~R->P&C_FLAG); \
  R->P&=~(N_FLAG|V_FLAG|Z_FLAG|C_FLAG); \
  R->P|=((R->A^Rg)&(R->A^K.B.l)&0x80? V_FLAG:0)| \
        (K.B.h? 0:C_FLAG)|ZNTable[K.B.l]; \
  R->A=K.B.l

#else /* NO_DECIMAL */

#define M_ADC(Rg) \
  if(R->P&D_FLAG) \
  { \
    K.B.l=(R->A&0x0F)+(Rg&0x0F)+(R->P&C_FLAG); \
    if(K.B.l>9) K.B.l+=6; \
    K.B.h=(R->A>>4)+(Rg>>4)+(K.B.l>15? 1:0); \
    R->A=(K.B.l&0x0F)|(K.B.h<<4); \
    R->P=(R->P&~C_FLAG)|(K.B.h>15? C_FLAG:0); \
  } \
  else \
  { \
    K.W=R->A+Rg+(R->P&C_FLAG); \
    R->P&=~(N_FLAG|V_FLAG|Z_FLAG|C_FLAG); \
    R->P|=(~(R->A^Rg)&(R->A^K.B.l)&0x80? V_FLAG:0)| \
          (K.B.h? C_FLAG:0)|ZNTable[K.B.l]; \
    R->A=K.B.l; \
  }

/* Warning! C_FLAG is inverted before SBC and after it */
#define M_SBC(Rg) \
  if(R->P&D_FLAG) \
  { \
    K.B.l=(R->A&0x0F)-(Rg&0x0F)-(~R->P&C_FLAG); \
    if(K.B.l&0x10) K.B.l-=6; \
    K.B.h=(R->A>>4)-(Rg>>4)-((K.B.l&0x10)>>4); \
    if(K.B.h&0x10) K.B.h-=6; \
    R->A=(K.B.l&0x0F)|(K.B.h<<4); \
    R->P=(R->P&~C_FLAG)|(K.B.h>15? 0:C_FLAG); \
  } \
  else \
  { \
    K.W=R->A-Rg-(~R->P&C_FLAG); \
    R->P&=~(N_FLAG|V_FLAG|Z_FLAG|C_FLAG); \
    R->P|=((R->A^Rg)&(R->A^K.B.l)&0x80? V_FLAG:0)| \
          (K.B.h? 0:C_FLAG)|ZNTable[K.B.l]; \
    R->A=K.B.l; \
  }

#endif /* NO_DECIMAL */

#define M_CMP(Rg1,Rg2) \
  K.W=Rg1-Rg2; \
  R->P&=~(N_FLAG|Z_FLAG|C_FLAG); \
  R->P|=ZNTable[K.B.l]|(K.B.h? 0:C_FLAG)
#define M_BIT(Rg) \
  R->P&=~(N_FLAG|V_FLAG|Z_FLAG); \
  R->P|=(Rg&(N_FLAG|V_FLAG))|(Rg&R->A? 0:Z_FLAG)

#define M_AND(Rg)	R->A&=Rg;M_FL(R->A)
#define M_ORA(Rg)	R->A|=Rg;M_FL(R->A)
#define M_EOR(Rg)	R->A^=Rg;M_FL(R->A)
#define M_INC(Rg)	Rg++;M_FL(Rg)
#define M_DEC(Rg)	Rg--;M_FL(Rg)

#define M_ASL(Rg)	R->P&=~C_FLAG;R->P|=Rg>>7;Rg<<=1;M_FL(Rg)
#define M_LSR(Rg)	R->P&=~C_FLAG;R->P|=Rg&C_FLAG;Rg>>=1;M_FL(Rg)
#define M_ROL(Rg)	K.B.l=(Rg<<1)|(R->P&C_FLAG); \
			R->P&=~C_FLAG;R->P|=Rg>>7;Rg=K.B.l; \
			M_FL(Rg)
#define M_ROR(Rg)	K.B.l=(Rg>>1)|(R->P<<7); \
			R->P&=~C_FLAG;R->P|=Rg&C_FLAG;Rg=K.B.l; \
			M_FL(Rg)

/** Reset6502() **********************************************/
/** This function can be used to reset the registers before **/
/** starting execution with Run6502(). It sets registers to **/
/** their initial values.                                   **/
/*************************************************************/
void Reset6502(M6502 *R)
{
  R->A=R->X=R->Y=0x00;
  R->P=Z_FLAG|R_FLAG;
  R->S=0xFF;
  R->PC.B.l=Rd6502(0xFFFC);
  R->PC.B.h=Rd6502(0xFFFD);   
  R->ICount=R->IPeriod;
  R->IRequest=INT_NONE;
  R->AfterCLI=0;
}

/** Exec6502() ***********************************************/
/** This function will execute a single 6502 opcode. It     **/
/** will then return next PC, and current register values   **/
/** in R.                                                   **/
/*************************************************************/
#ifdef EXEC6502
int Exec6502(M6502 *R,int RunCycles)
{
  register pair J,K;
  register byte I;

  /* Execute requested number of cycles */
  while(RunCycles>0)
  {
#ifdef DEBUG
    /* Turn tracing on when reached trap address */
    if(R->PC.W==R->Trap) R->Trace=1;
    /* Call single-step debugger, exit if requested */
    if(R->Trace)
      if(!Debug6502(R)) return(RunCycles);
#endif

    I=Op6502(R->PC.W++);
    RunCycles-=Cycles[I];
    switch(I)
    {
#include "Codes.h"
    }
  }

  /* Return number of cycles left (<=0) */
  return(RunCycles);
}
#endif /* EXEC6502 */

/** Int6502() ************************************************/
/** This function will generate interrupt of a given type.  **/
/** INT_NMI will cause a non-maskable interrupt. INT_IRQ    **/
/** will cause a normal interrupt, unless I_FLAG set in R.  **/
/*************************************************************/
void Int6502(M6502 *R,byte Type)
{
  register pair J;

  if((Type==INT_NMI)||((Type==INT_IRQ)&&!(R->P&I_FLAG)))
  {
    R->ICount-=7;
    M_PUSH(R->PC.B.h);
    M_PUSH(R->PC.B.l);
    M_PUSH(R->P&~B_FLAG);
    R->P&=~D_FLAG;
    if(R->IAutoReset&&(Type==R->IRequest)) R->IRequest=INT_NONE;
    if(Type==INT_NMI) J.W=0xFFFA; else { R->P|=I_FLAG;J.W=0xFFFE; }
    R->PC.B.l=Rd6502(J.W++);
    R->PC.B.h=Rd6502(J.W);
  }
}

/** Run6502() ************************************************/
/** This function will run 6502 code until Loop6502() call  **/
/** returns INT_QUIT. It will return the PC at which        **/
/** emulation stopped, and current register values in R.    **/
/*************************************************************/
#ifndef EXEC6502
word Run6502(M6502 *R)
{
  register pair J,K;
  register byte I;

  for(;;)
  {
#ifdef S60
    Opt6502(R);
#else /* !S60 */

#ifdef DEBUG
    /* Turn tracing on when reached trap address */
    if(R->PC.W==R->Trap) R->Trace=1;
    /* Call single-step debugger, exit if requested */
    if(R->Trace)
      if(!Debug6502(R)) return(R->PC.W);
#endif

    I=Op6502(R->PC.W++);
    R->ICount-=Cycles[I];
    switch(I)
    {
#include "Codes.h"
    }

#endif /* !S60 */

    /* If cycle counter expired... */
    if(R->ICount<=0)
    {
      /* If we have come after CLI, get INT_? from IRequest */
      /* Otherwise, get it from the loop handler            */
      if(R->AfterCLI)
      {
        I=R->IRequest;            /* Get pending interrupt     */
        R->ICount+=R->IBackup-1;  /* Restore the ICount        */
        R->AfterCLI=0;            /* Done with AfterCLI state  */
      }
      else
      {
        I=Loop6502(R);            /* Call the periodic handler */
        R->ICount+=R->IPeriod;    /* Reset the cycle counter   */
        if(!I) I=R->IRequest;     /* Realize pending interrupt */
      }

      if(I==INT_QUIT) return(R->PC.W); /* Exit if INT_QUIT     */
      if(I) Int6502(R,I);              /* Interrupt if needed  */ 
    }
  }

  /* Execution stopped */
  return(R->PC.W);
}
#endif /* !EXEC6502 */
