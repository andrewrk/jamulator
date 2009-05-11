/** M6502: portable 6502 emulator ****************************/
/**                                                         **/
/**                       ConDebug.c                        **/
/**                                                         **/
/** This file contains a console version of the built-in    **/
/** debugger, using EMULib's Console.c. When -DCONDEBUG is  **/
/** ommitted, ConDebug.c just includes the default command  **/
/** line based debugger (Debug.c).                          **/
/**                                                         **/
/** Copyright (C) Marat Fayzullin 2005-2007                 **/
/**     You are not allowed to distribute this software     **/
/**     commercially. Please, notify me, if you make any    **/   
/**     changes to this file.                               **/
/*************************************************************/
#ifdef DEBUG

#ifndef CONDEBUG
/** Normal Debug6502() ***************************************/
/** When CONDEBUG #undefined, we use plain command line.    **/
/*************************************************************/
#include "Debug.c"

#else
/** Console Debug6502() **************************************/
/** When CONDEBUG #defined, we use EMULib console.          **/
/*************************************************************/

#include "M6502.h"
#include "Console.h"
#include <stdlib.h>

#define Debug6502 OriginalDebug6502
#include "Debug.c"
#undef Debug6502

#define CLR_BACK   PIXEL(255,255,255)
#define CLR_TEXT   PIXEL(0,0,0)
#define CLR_DIALOG PIXEL(0,100,0)
#define CLR_PC     PIXEL(255,0,0)
#define CLR_SP     PIXEL(0,0,100)

static byte ChrDump(byte C)
{
  return((C>=32)&&(C<128)? C:'.');
}
 
/** Debug6502() **********************************************/
/** This function should exist if DEBUG is #defined. When   **/
/** Trace!=0, it is called after each command executed by   **/
/** the CPU, and given the 6502 registers.                  **/
/*************************************************************/
byte Debug6502(M6502 *R)
{
  char S[1024];
  word A,Addr,ABuf[20];
  int J,I,K,X,Y,MemoryDump,DrawWindow,ExitNow;

  /* If we don't have enough screen estate... */
  if((VideoW<32*8)||(VideoH<23*8))
  {
    /* Show warning message */
    CONMsg(
      -1,-1,-1,-1,PIXEL(255,255,255),PIXEL(255,0,0),
      "Error","Screen is\0too small!\0\0"
    );
    /* Continue emulation */
    return(1);
  }

  X    = ((VideoW>>3)-30)>>1;
  Y    = ((VideoH>>3)-25)>>1;
  Addr = R->PC.W;
  A    = ~Addr;
  K    = 0;
  
  for(DrawWindow=1,MemoryDump=ExitNow=0;!ExitNow&&VideoImg;)
  {
    if(DrawWindow)
    {
      CONWindow(X,Y,30,25,CLR_TEXT,CLR_BACK,"6502 Debugger");

      sprintf(S,"PC:%04X",R->PC.W);
      CONSetColor(CLR_BACK,CLR_PC);
      CONPrint(X+1,Y+2,S);
      sprintf(S,"SP:%04X",R->S+0x100);
      CONSetColor(CLR_BACK,CLR_SP);
      CONPrint(X+9,Y+2,S);
      CONPrint(X+25,Y+8,"STCK");

      sprintf(S,"P:[%c%c%c%c%c%c%c%c]",
        R->P&0x80? 'N':'.',
        R->P&0x40? 'V':'.',
        R->P&0x20? 'R':'.',
        R->P&0x10? 'B':'.',
        R->P&0x08? 'D':'.',
        R->P&0x04? 'I':'.',
        R->P&0x02? 'Z':'.',
        R->P&0x01? 'C':'.'
      );
      CONSetColor(CLR_BACK,CLR_DIALOG);
      CONPrint(X+17,Y+2,S);

      sprintf(S,"A:%02X\nX:%02X\nY:%02X",R->A,R->X,R->Y);
      CONSetColor(CLR_TEXT,CLR_BACK);
      CONPrint(X+25,Y+4,S);

      for(J=0;J<15;++J)
      {
        sprintf(S,"%02X",Rd6502(0x0100+(byte)(R->S+J+1)));
        CONPrint(X+26,Y+J+9,S);
      }

      DrawWindow=0;
      A=~Addr;
    }

    /* If top address has changed... */
    if(A!=Addr)
    {
      /* Clear display */
      CONBox((X+1)<<3,(Y+4)<<3,23*8,20*8,CLR_BACK);

      if(MemoryDump)
      {
        /* Draw memory dump */
        for(J=0,A=Addr;J<20;J++,A+=4)
        {
          if(A==R->PC.W)         CONSetColor(CLR_BACK,CLR_PC);
          else if(A==R->S+0x100) CONSetColor(CLR_BACK,CLR_SP);
          else                   CONSetColor(CLR_TEXT,CLR_BACK);
          sprintf(S,"%04X%c",A,A==R->PC.W? CON_MORE:A==R->S+0x100? CON_LESS:':');
          CONPrint(X+1,Y+J+4,S);

          CONSetColor(CLR_TEXT,CLR_BACK);
          sprintf(S,
            "%02X %02X %02X %02X %c%c%c%c",
            Rd6502(A),Rd6502(A+1),Rd6502(A+2),Rd6502(A+3),
            ChrDump(Rd6502(A)),ChrDump(Rd6502(A+1)),
            ChrDump(Rd6502(A+2)),ChrDump(Rd6502(A+3))
          );
          CONPrint(X+7,Y+J+4,S);
        }
      }
      else
      {
        /* Draw listing */
        for(J=0,A=Addr;J<20;J++)
        {
          if(A==R->PC.W)         CONSetColor(CLR_BACK,CLR_PC);
          else if(A==R->S+0x100) CONSetColor(CLR_BACK,CLR_SP);
          else                   CONSetColor(CLR_TEXT,CLR_BACK);
          sprintf(S,"%04X%c",A,A==R->PC.W? CON_MORE:A==R->S+0x100? CON_LESS:':');
          CONPrint(X+1,Y+J+4,S);

          ABuf[J]=A;
          A+=DAsm(S,A);

          CONSetColor(CLR_TEXT,CLR_BACK);
          CONPrintN(X+7,Y+J+4,S,17);
        }
      }

      /* Display redrawn */
      A=Addr;
    }

    /* Draw pointer */
    CONChar(X+6,Y+K+4,CON_ARROW);

    /* Show screen buffer */
    ShowVideo();

    /* Get key code */
    I=WaitKey();

    /* Clear pointer */
    CONChar(X+6,Y+K+4,' ');

    /* Get and process key code */
    switch(I)
    {
      case 'H':
        CONMsg(
          -1,-1,-1,-1,
          CLR_BACK,CLR_DIALOG,
          "Debugger Help",
          "ENTER - Execute next opcode\0"
          "   UP - Previous opcode\0"
          " DOWN - Next opcode\0"
          " LEFT - Page up\0"
          "RIGHT - Page down\0"
          "    H - This help page\0"
          "    G - Go to address\0"
          "    D - Disassembler view\0"
          "    M - Memory dump view\0"
          "    S - Show stack\0"
          "    J - Jump to cursor\0"
          "    R - Run to cursor\0"
          "    C - Continue execution\0"
          "    Q - Quit emulator\0"
        );
        DrawWindow=1;
        break;
      case CON_UP:
        if(K) --K;
        else
          if(MemoryDump) Addr-=4;
          else for(--Addr;Addr+DAsm(S,Addr)>A;--Addr);
        break;
      case CON_DOWN:
        if(K<19) ++K;
        else
          if(MemoryDump) Addr+=4;
          else Addr+=DAsm(S,Addr);
        break;
      case CON_LEFT:
        if(MemoryDump)
          Addr-=4*20;
        else
        {
          for(I=20,Addr=~A;(Addr>A)||((A^Addr)&~Addr&0x8000);++I)
            for(J=0,Addr=A-I;J<20;++J) Addr+=DAsm(S,Addr);
          Addr=A-I+1;
        }
        break; 
      case CON_RIGHT:
        if(MemoryDump)
          Addr+=4*20;
        else
          for(J=0;J<20;++J) Addr+=DAsm(S,Addr);
        break;
      case CON_OK:
        ExitNow=1;
        break;
      case 'Q':
        return(0);
      case CON_EXIT:
      case 'C':
        R->Trap=0xFFFF;
        R->Trace=0;
        ExitNow=1;
        break;
      case 'R':
        R->Trap=ABuf[K];
        R->Trace=0;
        ExitNow=1;
        break;
      case 'M':
        MemoryDump=1;
        A=~Addr;
        break;
      case 'S':
        MemoryDump=1;
        Addr=R->S+0x100;
        K=0;
        A=~Addr;
        break;
      case 'D':
        MemoryDump=0;
        A=~Addr;
        break;        
      case 'G':
        if(CONInput(-1,-1,CLR_BACK,CLR_DIALOG,"Go to Address:",S,5|CON_HEX))
        { Addr=strtol(S,0,16);K=0; }
        DrawWindow=1;
        break;
      case 'J':
        R->PC.W=ABuf[K];
        A=~Addr;
        break;
    }
  }

  /* Continue emulation */
  return(1);
}

#endif /* CONDEBUG */
#endif /* DEBUG */
