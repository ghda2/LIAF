/* LIAF C runtime. Values are statically checked before C emission.
   Allocations have process lifetime. This backend currently targets CLI programs. */
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <limits.h>
#include <stdarg.h>
#include <math.h>
#ifdef _WIN32
#include <windows.h>
#else
#include <pthread.h>
#include <unistd.h>
#endif

typedef struct V { int tag; int64_t i; double f; char *s; void *p; size_t len; } V;
typedef struct List { size_t n,cap; V *a; } List;
typedef struct Pair { V key,value; } Pair;
typedef struct Map { size_t n,cap; Pair *a; } Map;
typedef struct Result { int ok; V value; } Result;
typedef struct Option { int some; V value; } Option;
typedef struct Object { size_t n; const char **names; V *values; } Object;
enum { LIAF_NIL, LIAF_INTEGER, LIAF_FLOATING, LIAF_BOOLEAN, LIAF_STRING, LIAF_LIST, LIAF_MAP, LIAF_RESULT, LIAF_OBJECT, LIAF_CHANNEL, LIAF_OPTION };
static void *mem(size_t n) { void *p=calloc(1,n?n:1); if(!p){fputs("LIAF: out of memory\n",stderr);exit(2);} return p; }
static V vi(int64_t i){V v={0};v.tag=LIAF_INTEGER;v.i=i;return v;}
static V vf(double f){V v={0};v.tag=LIAF_FLOATING;v.f=f;return v;}
static V vb(int b){V v=vi(b!=0);v.tag=LIAF_BOOLEAN;return v;}
static V vsn(const char *s,size_t n){V v={0};v.tag=LIAF_STRING;v.s=mem(n+1);memcpy(v.s,s,n);v.len=n;return v;}
static V vs(const char *s){return vsn(s,strlen(s));}
static V vp(int tag,void*p){V v={0};v.tag=tag;v.p=p;return v;}
static V vr(int ok,V value){Result*r=mem(sizeof(*r));r->ok=ok;r->value=value;return vp(LIAF_RESULT,r);}
static V verr(const char*s){return vr(0,vs(s));}
static V vsome(V v){Option*o=mem(sizeof(*o));o->some=1;o->value=v;return vp(LIAF_OPTION,o);}
static V voption_none(void){Option*o=mem(sizeof(*o));o->some=0;return vp(LIAF_OPTION,o);}
static V vnone(void){V v={0};return v;}
static int eq(V a,V b){if(a.tag!=b.tag)return 0;switch(a.tag){case LIAF_STRING:return a.len==b.len&&!memcmp(a.s,b.s,a.len);case LIAF_FLOATING:return a.f==b.f;default:return a.i==b.i;}}
static V vintstr(V n){char b[64];snprintf(b,sizeof(b),"%lld",(long long)n.i);return vs(b);}
static V vtext(V v){char b[128];switch(v.tag){case LIAF_STRING:return v;case LIAF_INTEGER:return vintstr(v);case LIAF_BOOLEAN:return vs(v.i?"true":"false");case LIAF_FLOATING:snprintf(b,sizeof(b),"%.17g",v.f);return vs(b);default:return vs("");}}
static V vprint(int count,V *args,int newline){int i;for(i=0;i<count;i++){V s=vtext(args[i]);if(newline&&i)fputc(' ',stdout);fwrite(s.s,1,s.len,stdout);}if(newline)fputc('\n',stdout);fflush(stdout);return vnone();}
static V vconcat(int count,V*args){size_t size=0,pos=0;int i;V *text=mem(sizeof(V)*(count?count:1));for(i=0;i<count;i++){text[i]=vtext(args[i]);size+=text[i].len;}char *s=mem(size+1);for(i=0;i<count;i++){memcpy(s+pos,text[i].s,text[i].len);pos+=text[i].len;}V v=vsn(s,size);free(s);free(text);return v;}
static V vunwrap_or(V r, V fallback) {
    if (r.tag == LIAF_RESULT) {
        Result *res = (Result *)r.p;
        return res->ok ? res->value : fallback;
    }
    if (r.tag == LIAF_OPTION) {
        Option *opt = (Option *)r.p;
        return opt->some ? opt->value : fallback;
    }
    return fallback;
}
static V vfmt(V pattern, int count, V *args) {
    if (pattern.tag != LIAF_STRING) return vs("");
    size_t cap = pattern.len * 2 + 64;
    char *buf = mem(cap);
    size_t len = 0;
    int arg_idx = 0;
    for (size_t i = 0; i < pattern.len; ) {
        if (i + 1 < pattern.len && pattern.s[i] == '{' && pattern.s[i+1] == '{') {
            if (len + 1 >= cap) {
                cap *= 2;
                char *nb = mem(cap);
                memcpy(nb, buf, len);
                free(buf);
                buf = nb;
            }
            buf[len++] = '{';
            i += 2;
        } else if (i + 1 < pattern.len && pattern.s[i] == '}' && pattern.s[i+1] == '}') {
            if (len + 1 >= cap) {
                cap *= 2;
                char *nb = mem(cap);
                memcpy(nb, buf, len);
                free(buf);
                buf = nb;
            }
            buf[len++] = '}';
            i += 2;
        } else if (i + 1 < pattern.len && pattern.s[i] == '{' && pattern.s[i+1] == '}') {
            if (arg_idx < count) {
                V val_str = vtext(args[arg_idx++]);
                while (len + val_str.len + 1 >= cap) {
                    cap *= 2;
                    char *nb = mem(cap);
                    memcpy(nb, buf, len);
                    free(buf);
                    buf = nb;
                }
                memcpy(buf + len, val_str.s, val_str.len);
                len += val_str.len;
            } else {
                if (len + 2 >= cap) {
                    cap *= 2;
                    char *nb = mem(cap);
                    memcpy(nb, buf, len);
                    free(buf);
                    buf = nb;
                }
                buf[len++] = '{';
                buf[len++] = '}';
            }
            i += 2;
        } else {
            if (len + 1 >= cap) {
                cap *= 2;
                char *nb = mem(cap);
                memcpy(nb, buf, len);
                free(buf);
                buf = nb;
            }
            buf[len++] = pattern.s[i++];
        }
    }
    V res = vsn(buf, len);
    free(buf);
    return res;
}
static V vlist(void){return vp(LIAF_LIST,mem(sizeof(List)));}
static V vpush(V l,V value){List*p=l.p;if(p->n==p->cap){size_t cap=p->cap?p->cap*2:8;V*a=mem(cap*sizeof(V));if(p->a){memcpy(a,p->a,p->n*sizeof(V));free(p->a);}p->a=a;p->cap=cap;}p->a[p->n++]=value;return vnone();}
static V vlist_literal(size_t n, V *items){V l=vlist();for(size_t i=0;i<n;i++)vpush(l,items[i]);return l;}
static V vget(V l,V i){List*p=l.p;if(i.i<0||(uint64_t)i.i>=p->n)return verr("list index out of bounds");return vr(1,p->a[i.i]);}
static V vset(V l,V i,V value){List*p=l.p;if(i.i<0||(uint64_t)i.i>=p->n)return verr("list index out of bounds");p->a[i.i]=value;return vr(1,vb(1));}
static V vlistremove(V l, V i){List*p=l.p;if(i.i<0||(size_t)i.i>=p->n)return verr("list index out of bounds");for(size_t j=(size_t)i.i;j+1<p->n;j++)p->a[j]=p->a[j+1];p->n--;return vr(1,vb(1));}
static V vlistpop(V l){List*p=l.p;if(p->n==0)return verr("list is empty");V val=p->a[--p->n];return vr(1,val);}
static int vcmp(const void *a, const void *b){const V *va=a,*vb=b;if(va->tag==LIAF_INTEGER)return (va->i>vb->i)-(va->i<vb->i);if(va->tag==LIAF_FLOATING)return (va->f>vb->f)-(va->f<vb->f);if(va->tag==LIAF_STRING)return strcmp(va->s,vb->s);return 0;}
static V vlistsort(V l){List*p=l.p;if(p->n>1)qsort(p->a,p->n,sizeof(V),vcmp);return vnone();}
static V vlistcontains(V l, V x){List*p=l.p;for(size_t i=0;i<p->n;i++)if(eq(p->a[i],x))return vb(1);return vb(0);}
static V vmap(void){return vp(LIAF_MAP,mem(sizeof(Map)));}
static V vmset(V m,V k,V value){Map*p=m.p;size_t i;for(i=0;i<p->n;i++)if(eq(p->a[i].key,k)){p->a[i].value=value;return vnone();}if(p->n==p->cap){size_t cap=p->cap?p->cap*2:8;Pair*a=mem(cap*sizeof(Pair));if(p->a){memcpy(a,p->a,p->n*sizeof(Pair));free(p->a);}p->a=a;p->cap=cap;}p->a[p->n].key=k;p->a[p->n++].value=value;return vnone();}
static V vmdelete(V m, V k){Map*p=m.p;for(size_t i=0;i<p->n;i++)if(eq(p->a[i].key,k)){for(size_t j=i;j+1<p->n;j++)p->a[j]=p->a[j+1];p->n--;break;}return vnone();}
static V vmkeys(V m){Map*p=m.p;V list=vlist();for(size_t i=0;i<p->n;i++)vpush(list,p->a[i].key);vlistsort(list);return list;}
static V vmget(V m,V k,int has){Map*p=m.p;size_t i;for(i=0;i<p->n;i++)if(eq(p->a[i].key,k))return has?vb(1):vr(1,p->a[i].value);return has?vb(0):verr("map key not found");}
static V vread(V path){FILE*f=fopen(path.s,"rb");char*b;long n;size_t got;if(!f)return verr(strerror(errno));if(fseek(f,0,SEEK_END)!=0||(n=ftell(f))<0){fclose(f);return verr("Could not determine file size");}rewind(f);b=mem((size_t)n+1);got=fread(b,1,(size_t)n,f);if(ferror(f)){fclose(f);free(b);return verr("File read failed");}fclose(f);V v=vsn(b,got);free(b);return vr(1,v);}
static V vwrite(V path,V s){FILE*f=fopen(path.s,"wb");size_t n;int closed;if(!f)return verr(strerror(errno));n=fwrite(s.s,1,s.len,f);closed=fclose(f);if(n!=s.len||closed)return verr("File write failed");return vr(1,vb(1));}
static V vremove(V path){if(remove(path.s))return verr(strerror(errno));return vr(1,vb(1));}
static V vexists(V path){FILE*f=fopen(path.s,"rb");if(f){fclose(f);return vb(1);}return vb(0);}
static V vintparse(V s){char*end;long long n;errno=0;n=strtoll(s.s,&end,0);if(errno||end!=s.s+s.len||end==s.s){char b[256];snprintf(b,sizeof(b),"int-from-str: \"%s\" nao e um inteiro valido",s.s);return verr(b);}return vr(1,vi(n));}
static V vfloatparse(V s){char*end;double n;errno=0;n=strtod(s.s,&end);if(errno||end!=s.s+s.len||end==s.s){char b[256];snprintf(b,sizeof(b),"float-from-str: \"%s\" nao e um float valido",s.s);return verr(b);}return vr(1,vf(n));}
static V vboolparse(V s){if(!strcmp(s.s,"true"))return vr(1,vb(1));if(!strcmp(s.s,"false"))return vr(1,vb(0));char b[256];snprintf(b,sizeof(b),"bool-from-str: \"%s\" nao e um booleano valido",s.s);return verr(b);}
static V vstrfloat(V f){char b[128];snprintf(b,sizeof(b),"%.17g",f.f);return vs(b);}
static V vstrbool(V b){return vs(b.i?"true":"false");}
static int64_t utf8_len(const char *s, size_t len) {
    int64_t count = 0;
    for (size_t i = 0; i < len; i++) {
        if (((unsigned char)s[i] & 0xC0) != 0x80) count++;
    }
    return count;
}
static size_t utf8_rune_offset(const char *s, size_t len, int64_t rune_idx) {
    int64_t cur = 0;
    for (size_t i = 0; i < len; i++) {
        if (((unsigned char)s[i] & 0xC0) != 0x80) {
            if (cur == rune_idx) return i;
            cur++;
        }
    }
    return len;
}
static V vstrlen(V s){return vi(utf8_len(s.s, s.len));}
static V vstrbytelen(V s){return vi(s.len);}
static V vslice(V s,V a,V b){
    int64_t total = utf8_len(s.s, s.len);
    if(a.i<0 || b.i<a.i || b.i>total) return verr("string range out of bounds");
    size_t off1 = utf8_rune_offset(s.s, s.len, a.i);
    size_t off2 = utf8_rune_offset(s.s, s.len, b.i);
    return vr(1, vsn(s.s+off1, off2-off1));
}
static V vstrget(V s,V idx){
    int64_t total = utf8_len(s.s, s.len);
    if(idx.i<0 || idx.i>=total) return verr("string index out of bounds");
    size_t off1 = utf8_rune_offset(s.s, s.len, idx.i);
    size_t off2 = utf8_rune_offset(s.s, s.len, idx.i+1);
    return vr(1, vsn(s.s+off1, off2-off1));
}
static V vstrcontains(V s,V sub){return vb(strstr(s.s, sub.s)!=NULL);}
static V vstrstartswith(V s,V p){if(p.len>s.len)return vb(0);return vb(memcmp(s.s, p.s, p.len)==0);}
static V vstrendswith(V s,V p){if(p.len>s.len)return vb(0);return vb(memcmp(s.s+s.len-p.len, p.s, p.len)==0);}
static V vstrindex(V s,V sub){
    char* p = strstr(s.s, sub.s);
    if(!p) return vi(-1);
    return vi(utf8_len(s.s, (size_t)(p - s.s)));
}
static V vstrtrim(V s){
    size_t start = 0;
    while(start < s.len && isspace((unsigned char)s.s[start])) start++;
    size_t end = s.len;
    while(end > start && isspace((unsigned char)s.s[end-1])) end--;
    return vsn(s.s+start, end-start);
}
static V vstrreplace(V s,V old,V nstr){
    if(old.len==0) return s;
    size_t count = 0;
    const char* p = s.s;
    while((p = strstr(p, old.s)) != NULL){count++; p += old.len;}
    if(count == 0) return s;
    size_t new_len = s.len + count*(nstr.len - old.len);
    char* buf = mem(new_len + 1);
    char* dst = buf;
    const char* src = s.s;
    while((p = strstr(src, old.s)) != NULL){
        size_t seg = (size_t)(p - src);
        memcpy(dst, src, seg);
        dst += seg;
        memcpy(dst, nstr.s, nstr.len);
        dst += nstr.len;
        src = p + old.len;
    }
    size_t rest = (size_t)(s.s + s.len - src);
    memcpy(dst, src, rest);
    dst[rest] = '\0';
    V res = vsn(buf, new_len);
    free(buf);
    return res;
}
static V vstrupper(V s){
    char* buf = mem(s.len + 1);
    for(size_t i = 0; i < s.len; i++){
        unsigned char c = (unsigned char)s.s[i];
        if(c == 0xC3 && i + 1 < s.len){
            unsigned char c2 = (unsigned char)s.s[i+1];
            if(c2 >= 0xA0 && c2 <= 0xBE && c2 != 0xB7){
                buf[i] = (char)0xC3;
                buf[i+1] = (char)(c2 - 0x20);
                i++;
                continue;
            }
        }
        buf[i] = (char)toupper(c);
    }
    buf[s.len] = '\0';
    V res = vsn(buf, s.len);
    free(buf);
    return res;
}
static V vstrlower(V s){
    char* buf = mem(s.len + 1);
    for(size_t i = 0; i < s.len; i++){
        unsigned char c = (unsigned char)s.s[i];
        if(c == 0xC3 && i + 1 < s.len){
            unsigned char c2 = (unsigned char)s.s[i+1];
            if(c2 >= 0x80 && c2 <= 0x9E && c2 != 0x97){
                buf[i] = (char)0xC3;
                buf[i+1] = (char)(c2 + 0x20);
                i++;
                continue;
            }
        }
        buf[i] = (char)tolower(c);
    }
    buf[s.len] = '\0';
    V res = vsn(buf, s.len);
    free(buf);
    return res;
}
static V vstrsplit(V s,V sep){
    V list = vlist();
    if(sep.len == 0){
        int64_t total = utf8_len(s.s, s.len);
        for(int64_t i = 0; i < total; i++){
            size_t o1 = utf8_rune_offset(s.s, s.len, i);
            size_t o2 = utf8_rune_offset(s.s, s.len, i + 1);
            vpush(list, vsn(s.s + o1, o2 - o1));
        }
        return list;
    }
    const char* src = s.s;
    const char* p;
    while((p = strstr(src, sep.s)) != NULL){
        vpush(list, vsn(src, (size_t)(p - src)));
        src = p + sep.len;
    }
    vpush(list, vsn(src, (size_t)(s.s + s.len - src)));
    return list;
}
static V vstrjoin(V list,V sep){
    List* p = list.p;
    if(p->n == 0) return vs("");
    size_t total = 0;
    for(size_t i = 0; i < p->n; i++){
        total += p->a[i].len;
        if(i + 1 < p->n) total += sep.len;
    }
    char* buf = mem(total + 1);
    char* dst = buf;
    for(size_t i = 0; i < p->n; i++){
        memcpy(dst, p->a[i].s, p->a[i].len);
        dst += p->a[i].len;
        if(i + 1 < p->n){
            memcpy(dst, sep.s, sep.len);
            dst += sep.len;
        }
    }
    buf[total] = '\0';
    V res = vsn(buf, total);
    free(buf);
    return res;
}
static V vobject(int n,const char**names,V*values){Object*p=mem(sizeof(*p));p->n=n;p->names=mem(sizeof(char*)*(n?n:1));p->values=mem(sizeof(V)*(n?n:1));memcpy(p->names,names,n*sizeof(char*));memcpy(p->values,values,n*sizeof(V));return vp(LIAF_OBJECT,p);}
static V vfield(V obj,const char*name){Object*p=obj.p;size_t i;for(i=0;i<p->n;i++)if(!strcmp(p->names[i],name))return p->values[i];return vnone();}
static inline int check_add_overflow(int64_t a, int64_t b, int64_t *res) {
#if defined(__GNUC__) || defined(__clang__)
    return __builtin_add_overflow(a, b, res);
#else
    if ((b > 0 && a > INT64_MAX - b) || (b < 0 && a < INT64_MIN - b)) return 1;
    *res = a + b;
    return 0;
#endif
}

static inline int check_sub_overflow(int64_t a, int64_t b, int64_t *res) {
#if defined(__GNUC__) || defined(__clang__)
    return __builtin_sub_overflow(a, b, res);
#else
    if ((b < 0 && a > INT64_MAX + b) || (b > 0 && a < INT64_MIN + b)) return 1;
    *res = a - b;
    return 0;
#endif
}

static inline int check_mul_overflow(int64_t a, int64_t b, int64_t *res) {
#if defined(__GNUC__) || defined(__clang__)
    return __builtin_mul_overflow(a, b, res);
#else
    if (a == 0 || b == 0) { *res = 0; return 0; }
    if ((a == -1 && b == INT64_MIN) || (b == -1 && a == INT64_MIN)) return 1;
    *res = a * b;
    return (*res / b != a);
#endif
}

static V vadd(V a, V b) {
    int64_t res;
    if (check_add_overflow(a.i, b.i, &res)) {
        fputs("liaf: overflow na adicao de inteiros\n", stderr);
        exit(1);
    }
    return vi(res);
}

static V vsub(V a, V b) {
    int64_t res;
    if (check_sub_overflow(a.i, b.i, &res)) {
        fputs("liaf: overflow na subtracao de inteiros\n", stderr);
        exit(1);
    }
    return vi(res);
}

static V vmul(V a, V b) {
    int64_t res;
    if (check_mul_overflow(a.i, b.i, &res)) {
        fputs("liaf: overflow na multiplicacao de inteiros\n", stderr);
        exit(1);
    }
    return vi(res);
}

static V vdiv(V a,V b){if(a.tag==LIAF_FLOATING)return vf(a.f/b.f);if(!b.i)return vr(0,vs("div: divisao por zero"));if(a.i==INT64_MIN&&b.i==-1){fputs("liaf: overflow na divisao de inteiros\n",stderr);exit(1);}return vr(1,vi(a.i/b.i));}
static V vmod(V a,V b){if(!b.i)return vr(0,vs("mod: divisao por zero"));if(a.i==INT64_MIN&&b.i==-1)return vr(1,vi(0));return vr(1,vi(a.i%b.i));}
static V vneg(V a){if(a.tag==LIAF_FLOATING)return vf(-a.f);if(a.i==INT64_MIN){fputs("liaf: overflow na negacao de inteiros\n",stderr);exit(1);}return vi(-a.i);}
static V vabs(V a){if(a.tag==LIAF_FLOATING)return vf(fabs(a.f));if(a.i==INT64_MIN){fputs("liaf: overflow no abs de inteiros\n",stderr);exit(1);}return vi(a.i<0?-a.i:a.i);}
static V vmin(V a,V b){if(a.tag==LIAF_FLOATING)return vf(a.f<b.f?a.f:b.f);return vi(a.i<b.i?a.i:b.i);}
static V vmax(V a,V b){if(a.tag==LIAF_FLOATING)return vf(a.f>b.f?a.f:b.f);return vi(a.i>b.i?a.i:b.i);}
static V vpow(V a,V b){if(a.tag==LIAF_FLOATING||b.tag==LIAF_FLOATING){double bf=a.tag==LIAF_FLOATING?a.f:(double)a.i;double ef=b.tag==LIAF_FLOATING?b.f:(double)b.i;return vf(pow(bf,ef));}int64_t base=a.i,exp=b.i,res=1;if(exp<0){fputs("liaf: expoente negativo em pow\n",stderr);exit(1);}while(exp>0){if(exp&1){if(check_mul_overflow(res,base,&res)){fputs("liaf: overflow na multiplicacao de inteiros\n",stderr);exit(1);}}if(exp>1){if(check_mul_overflow(base,base,&base)){fputs("liaf: overflow na multiplicacao de inteiros\n",stderr);exit(1);}}exp>>=1;}return vi(res);}
static V vsqrt(V a){return vf(sqrt(a.tag==LIAF_FLOATING?a.f:(double)a.i));}
static V vfloor(V a){return vf(floor(a.tag==LIAF_FLOATING?a.f:(double)a.i));}
static V vceil(V a){return vf(ceil(a.tag==LIAF_FLOATING?a.f:(double)a.i));}
static V vround(V a){return vf(round(a.tag==LIAF_FLOATING?a.f:(double)a.i));}

#ifdef _WIN32
typedef struct Channel{HANDLE writer,ready,consumed;V value;}Channel;
static V vchan(void){Channel*c=mem(sizeof(*c));c->writer=CreateSemaphoreA(0,1,1,0);c->ready=CreateSemaphoreA(0,0,1,0);c->consumed=CreateSemaphoreA(0,0,1,0);if(!c->writer||!c->ready||!c->consumed){fputs("channel allocation failed\n",stderr);exit(2);}return vp(LIAF_CHANNEL,c);}
static V vsend(V ch,V value){Channel*c=ch.p;WaitForSingleObject(c->writer,INFINITE);c->value=value;ReleaseSemaphore(c->ready,1,0);WaitForSingleObject(c->consumed,INFINITE);ReleaseSemaphore(c->writer,1,0);return vnone();}
static V vrecv(V ch){Channel*c=ch.p;V v;WaitForSingleObject(c->ready,INFINITE);v=c->value;ReleaseSemaphore(c->consumed,1,0);return v;}
static V vsleep(V ms){Sleep((DWORD)ms.i);return vnone();}
#else
typedef struct Channel{pthread_mutex_t lock;pthread_cond_t ready;int full;V value;}Channel;
static V vchan(void){Channel*c=mem(sizeof(*c));pthread_mutex_init(&c->lock,0);pthread_cond_init(&c->ready,0);return vp(LIAF_CHANNEL,c);}
static V vsend(V ch,V value){Channel*c=ch.p;pthread_mutex_lock(&c->lock);while(c->full)pthread_cond_wait(&c->ready,&c->lock);c->value=value;c->full=1;pthread_cond_broadcast(&c->ready);while(c->full)pthread_cond_wait(&c->ready,&c->lock);pthread_mutex_unlock(&c->lock);return vnone();}
static V vrecv(V ch){Channel*c=ch.p;V v;pthread_mutex_lock(&c->lock);while(!c->full)pthread_cond_wait(&c->ready,&c->lock);v=c->value;c->full=0;pthread_cond_broadcast(&c->ready);pthread_mutex_unlock(&c->lock);return v;}
static V vsleep(V ms){usleep((unsigned int)(ms.i*1000));return vnone();}
#endif
typedef V(*Worker)(V*);
typedef struct Work{Worker fn;V*args;}Work;
#ifdef _WIN32
static DWORD WINAPI vworker(LPVOID arg){Work*w=arg;w->fn(w->args);free(w->args);free(w);return 0;}
#else
static void*vworker(void*arg){Work*w=arg;w->fn(w->args);free(w->args);free(w);return 0;}
#endif
static V vspawn(Worker fn,int count,V*args){Work*w=mem(sizeof(*w));w->fn=fn;w->args=mem(sizeof(V)*(count?count:1));memcpy(w->args,args,sizeof(V)*count);
#ifdef _WIN32
HANDLE t=CreateThread(0,0,vworker,w,0,0);if(!t){fputs("thread creation failed\n",stderr);exit(2);}CloseHandle(t);
#else
pthread_t t;if(pthread_create(&t,0,vworker,w)){fputs("thread creation failed\n",stderr);exit(2);}pthread_detach(t);
#endif
return vnone();}
static V program_args;
