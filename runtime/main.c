#include "rom.h"
#include "assert.h"
#include "ppu.h"
#include "stdio.h"
#include "SDL/SDL.h"
#include "GL/glew.h"

typedef struct {
    SDL_Surface* screen;
    GLuint tex;
    bool pendingResize;
    int pendingResizeWidth;
    int pendingResizeHeight;
} Video;

static Video v;
static Ppu* p;
static int interruptRequested = ROM_INTERRUPT_NONE;
bool fast = false;

uint8_t *framebufferSlice = NULL;
int framebufferSize = 0;

typedef struct {
    uint64_t cycle;
    uint8_t padIndex;
    uint8_t btnIndex;
    uint8_t btnState;
} MovieFrame;
static char * movieFilename = NULL;
static MovieFrame* movie = NULL;
static uint64_t movieFrameCount;
static uint64_t frameIndex = 0;
static uint64_t cycleIndex = 0;

void loadMovie() {
    if (movieFilename == NULL) return;
    FILE *fd = fopen(movieFilename, "rb");
    if (fd == NULL) {
        perror("Error opening movie file");
        exit(1);
    }
    size_t n = fread(&movieFrameCount, 8, 1, fd);
    movie = malloc(sizeof(MovieFrame) * movieFrameCount);
    for (size_t i = 0; i < movieFrameCount; ++i) {
        n = fread(&movie[i].cycle, 8, 1, fd);
        n = fread(&movie[i].padIndex, 1, 1, fd);
        n = fread(&movie[i].btnIndex, 1, 1, fd);
        n = fread(&movie[i].btnState, 1, 1, fd);
    }
    if (ferror(fd) != 0) {
        perror("Error reading movie");
        exit(1);
    }
    fclose(fd);
}

void setPadState(SDLKey key, uint8_t value) {
    switch (key) {
        default: break; // to make warning go away
        case SDLK_2:
            rom_set_button_state(0, ROM_BUTTON_A, value);
            break;
        case SDLK_1:
            rom_set_button_state(0, ROM_BUTTON_B, value);
            break;
        case SDLK_RSHIFT:
            rom_set_button_state(0, ROM_BUTTON_SELECT, value);
            break;
        case SDLK_RETURN:
            rom_set_button_state(0, ROM_BUTTON_START, value);
            break;
        case SDLK_UP:
            rom_set_button_state(0, ROM_BUTTON_UP, value);
            break;
        case SDLK_DOWN:
            rom_set_button_state(0, ROM_BUTTON_DOWN, value);
            break;
        case SDLK_LEFT:
            rom_set_button_state(0, ROM_BUTTON_LEFT, value);
            break;
        case SDLK_RIGHT:
            rom_set_button_state(0, ROM_BUTTON_RIGHT, value);
            break;
    }
}

void setPadStateFromMovie() {
    if (movie == NULL) return;
    while (frameIndex < movieFrameCount && cycleIndex >= movie[frameIndex].cycle) {
        rom_set_button_state(
                movie[frameIndex].padIndex,
                movie[frameIndex].btnIndex,
                movie[frameIndex].btnState);
        frameIndex += 1;
    }
    if (frameIndex >= movieFrameCount) exit(0);
}

void flush_events() {
    SDL_Event event;

    while (SDL_PollEvent(&event)) {
        switch (event.type) {
        case SDL_VIDEORESIZE:
            v.pendingResize = true;
            v.pendingResizeWidth = event.resize.w;
            v.pendingResizeHeight = event.resize.h;
            break;
        case SDL_QUIT:
            exit(0);
        case SDL_KEYDOWN:
            setPadState(event.key.keysym.sym, ROM_PAD_STATE_ON);
            break;
        case SDL_KEYUP:
            setPadState(event.key.keysym.sym, ROM_PAD_STATE_OFF);
            break;
        }
    }
}

void step(uint8_t cycles) {
    cycleIndex += cycles;
    for (int i = 0; i < 3 * cycles; ++i) {
        Ppu_step(p);
    }
}

void rom_cycle(uint8_t cycles) {
    flush_events();
    setPadStateFromMovie();
    step(cycles);
    int req = interruptRequested;
    if (req != ROM_INTERRUPT_NONE) {
        interruptRequested = ROM_INTERRUPT_NONE;
        rom_start(req);
    }
}

void reshape_video(int width, int height) {
    int x_offset = 0;
    int y_offset = 0;

    double dWidth = width;
    double dHeight = height;
    double r = dHeight / dWidth;

    if (r > 0.9375) { // Height taller than ratio
        int h = 0.9375 * dWidth;
        y_offset = (height - h) / 2;
        height = h;
    } else if (r < 0.9375) { // Width wider
        double scrW, scrH;
        if (p->overscanEnabled) {
            scrW = 240.0;
            scrH = 224.0;
        } else {
            scrW = 256.0;
            scrH = 240.0;
        }

        int w = (scrH / scrW) * dHeight;
        x_offset = (width - w) / 2;
        width = w;
    }

    glViewport(x_offset, y_offset, width, height);
    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    glOrtho(-1, 1, -1, 1, -1, 1);
    glMatrixMode(GL_MODELVIEW);
    glLoadIdentity();
    glDisable(GL_DEPTH_TEST);
}

void init_video() {
    if (SDL_Init(SDL_INIT_VIDEO|SDL_INIT_AUDIO) != 0) {
        fprintf(stderr, "Unable to init SDL: %s\n", SDL_GetError());
        exit(1);
    }

    SDL_GL_SetAttribute(SDL_GL_SWAP_CONTROL, !fast); // vsync
    v.screen = SDL_SetVideoMode(512, 480, 32, SDL_OPENGL|SDL_RESIZABLE);

    if (v.screen == NULL) {
        fprintf(stderr, "Unable to set SDL video mode: %s\n", SDL_GetError());
        exit(1);
    }

    SDL_WM_SetCaption("jamulator", NULL);

    if (glewInit() != 0) {
        fprintf(stderr, "Unable to init glew\n");
        exit(1);
    }

    glEnable(GL_TEXTURE_2D);
    reshape_video(v.screen->w, v.screen->h);
    v.pendingResize = false;

    glGenTextures(1, &v.tex);
}

void vblankInterrupt() {
    interruptRequested = ROM_INTERRUPT_NMI;
}

void render() {
    if (v.pendingResize) {
        reshape_video(v.pendingResizeWidth, v.pendingResizeHeight);
        v.pendingResize = false;
    }
    if (framebufferSlice == NULL || framebufferSize != p->framebufferSize) {
        if (framebufferSlice != NULL) free(framebufferSlice);
        framebufferSlice = malloc(p->framebufferSize * 3);
        framebufferSize = p->framebufferSize;
    }
    for (int i = 0; i < p->framebufferSize; ++i) {
        framebufferSlice[i*3+0] = (p->framebuffer[i] >> 16) & 0xff;
        framebufferSlice[i*3+1] = (p->framebuffer[i] >> 8) & 0xff;
        framebufferSlice[i*3+2] = p->framebuffer[i] & 0xff;
    }

    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

    glBindTexture(GL_TEXTURE_2D, v.tex);

    int w = p->overscanEnabled ? 240 : 256;
    int h = p->overscanEnabled ? 224 : 240;
    glTexImage2D(GL_TEXTURE_2D, 0, 3, w, h, 0, GL_RGB, GL_UNSIGNED_BYTE, framebufferSlice);

    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);

    glBegin(GL_QUADS);
    glTexCoord2f(0.0, 1.0);
    glVertex3f(-1.0, -1.0, 0.0);
    glTexCoord2f(1.0, 1.0);
    glVertex3f(1.0, -1.0, 0.0);
    glTexCoord2f(1.0, 0.0);
    glVertex3f(1.0, 1.0, 0.0);
    glTexCoord2f(0.0, 0.0);
    glVertex3f(-1.0, 1.0, 0.0);
    glEnd();

    if (v.screen != NULL) {
        SDL_GL_SwapBuffers();
        SDL_Delay(0);
    }
}

void printUsage(char * command) {
    fprintf(stderr, "Usage:\n%s [-movie file] [-fast]\n", command);
    exit(1);
}

void parseFlags(int argc, char* argv[]) {
    for (int i = 1; i < argc; ++i) {
        char * arg = argv[i];
        if (arg[0] == '-') {
            if (strcmp(arg, "-movie") == 0 && i < argc - 1) {
                movieFilename = argv[i + 1];
                i += 1;
            } else if (strcmp(arg, "-fast") == 0) {
                fast = true;
            } else {
                printUsage(argv[0]);
            }
        } else {
            printUsage(argv[0]);
        }
    }
}

int main(int argc, char* argv[]) {
    parseFlags(argc, argv);
    loadMovie();
    p = Ppu_new();
    p->render = &render;
    p->vblankInterrupt = &vblankInterrupt;
    p->readRam = &rom_ram_read;
    Nametable_setMirroring(&p->nametables, rom_mirroring);
    assert(rom_chr_bank_count == 1);
    rom_read_chr(p->vram);
    init_video();
    rom_start(ROM_INTERRUPT_RESET);
    Ppu_dispose(p);
}

uint8_t rom_ppu_read_status() {
    return Ppu_readStatus(p);
}

uint8_t rom_ppu_read_oamdata(){
    return Ppu_readOamData(p);
}
uint8_t rom_ppu_read_data(){
    return Ppu_readData(p);
}

void rom_ppu_write_control(uint8_t b) {
    Ppu_writeControl(p, b);
}

void rom_ppu_write_mask(uint8_t b) {
    Ppu_writeMask(p, b);
}

void rom_ppu_write_oamaddress(uint8_t b) {
    Ppu_writeOamAddress(p, b);
}

void rom_ppu_write_address(uint8_t b) {
    Ppu_writeAddress(p, b);
}

void rom_ppu_write_data(uint8_t b) {
    Ppu_writeData(p, b);
}

void rom_ppu_write_oamdata(uint8_t b) {
    Ppu_writeOamData(p, b);
}

void rom_ppu_write_scroll(uint8_t b) {
    Ppu_writeScroll(p, b);
}
void rom_ppu_write_dma(uint8_t b) {
    Ppu_writeDma(p, b);

    // Halt the CPU for 512 cycles
    step(255);
    step(255);
    step(2);
}

uint8_t rom_apu_read_status() {
    return 0;
}
void rom_apu_write_square1control(uint8_t b){}
void rom_apu_write_square1sweeps(uint8_t b){}
void rom_apu_write_square1low(uint8_t b){}
void rom_apu_write_square1high(uint8_t b){}
void rom_apu_write_square2control(uint8_t b){}
void rom_apu_write_square2sweeps(uint8_t b){}
void rom_apu_write_square2low(uint8_t b){}
void rom_apu_write_square2high(uint8_t b){}
void rom_apu_write_trianglecontrol(uint8_t b){}
void rom_apu_write_trianglelow(uint8_t b){}
void rom_apu_write_trianglehigh(uint8_t b){}
void rom_apu_write_noisebase(uint8_t b){}
void rom_apu_write_noiseperiod(uint8_t b){}
void rom_apu_write_noiselength(uint8_t b){}
void rom_apu_write_dmcflags(uint8_t b){}
void rom_apu_write_dmcdirectload(uint8_t b){}
void rom_apu_write_dmcsampleaddress(uint8_t b){}
void rom_apu_write_dmcsamplelength(uint8_t b){}
void rom_apu_write_controlflags1(uint8_t b){}
void rom_apu_write_controlflags2(uint8_t b){}
