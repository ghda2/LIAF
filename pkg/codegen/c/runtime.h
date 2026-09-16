/* LIAF C runtime. Values are statically checked before C emission.
   Allocations have process lifetime. This backend currently targets CLI programs. */
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <limits.h>
#include <stdarg.h>
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
typedef struct Object { size_t n; const char **names; V *values; } Object;
enum { LIAF_NIL, LIAF_INTEGER, LIAF_FLOATING, LIAF_BOOLEAN, LIAF_STRING, LIAF_LIST, LIAF_MAP, LIAF_RESULT, LIAF_OBJECT, LIAF_CHANNEL };
static void *mem(size_t n) { void *p=calloc(1,n?n:1); if(!p){fputs("LIAF: out of memory\n",stderr);exit(2);} return p; }
static V vi(int64_t i){V v={0};v.tag=LIAF_INTEGER;v.i=i;return v;}
static V vf(double f){V v={0};v.tag=LIAF_FLOATING;v.f=f;return v;}
static V vb(int b){V v=vi(b!=0);v.tag=LIAF_BOOLEAN;return v;}
static V vsn(const char *s,size_t n){V v={0};v.tag=LIAF_STRING;v.s=mem(n+1);memcpy(v.s,s,n);v.len=n;return v;}
static V vs(const char *s){return vsn(s,strlen(s));}
static V vp(int tag,void*p){V v={0};v.tag=tag;v.p=p;return v;}
static V vr(int ok,V value){Result*r=mem(sizeof(*r));r->ok=ok;r->value=value;return vp(LIAF_RESULT,r);}
static V verr(const char*s){return vr(0,vs(s));}
static V vnone(void){V v={0};return v;}
static int eq(V a,V b){if(a.tag!=b.tag)return 0;switch(a.tag){case LIAF_STRING:return a.len==b.len&&!memcmp(a.s,b.s,a.len);case LIAF_FLOATING:return a.f==b.f;default:return a.i==b.i;}}
static V vintstr(V n){char b[64];snprintf(b,sizeof(b),"%lld",(long long)n.i);return vs(b);}
static V vtext(V v){char b[128];switch(v.tag){case LIAF_STRING:return v;case LIAF_INTEGER:return vintstr(v);case LIAF_BOOLEAN:return vs(v.i?"true":"false");case LIAF_FLOATING:snprintf(b,sizeof(b),"%.17g",v.f);return vs(b);default:return vs("");}}
static V vprint(int count,V *args,int newline){int i;for(i=0;i<count;i++){V s=vtext(args[i]);if(newline&&i)fputc(' ',stdout);fwrite(s.s,1,s.len,stdout);}if(newline)fputc('\n',stdout);fflush(stdout);return vnone();}
static V vconcat(int count,V*args){size_t size=0,pos=0;int i;V *text=mem(sizeof(V)*(count?count:1));for(i=0;i<count;i++){text[i]=vtext(args[i]);size+=text[i].len;}char *s=mem(size+1);for(i=0;i<count;i++){memcpy(s+pos,text[i].s,text[i].len);pos+=text[i].len;}V v=vsn(s,size);free(s);free(text);return v;}
static V vlist(void){return vp(LIAF_LIST,mem(sizeof(List)));}
static V vpush(V l,V value){List*p=l.p;if(p->n==p->cap){size_t cap=p->cap?p->cap*2:8;V*a=mem(cap*sizeof(V));if(p->a){memcpy(a,p->a,p->n*sizeof(V));free(p->a);}p->a=a;p->cap=cap;}p->a[p->n++]=value;return vnone();}
static V vget(V l,V i){List*p=l.p;if(i.i<0||(uint64_t)i.i>=p->n)return verr("list index out of bounds");return vr(1,p->a[i.i]);}
static V vset(V l,V i,V value){List*p=l.p;if(i.i<0||(uint64_t)i.i>=p->n)return verr("list index out of bounds");p->a[i.i]=value;return vr(1,vb(1));}
static V vmap(void){return vp(LIAF_MAP,mem(sizeof(Map)));}
static V vmset(V m,V k,V value){Map*p=m.p;size_t i;for(i=0;i<p->n;i++)if(eq(p->a[i].key,k)){p->a[i].value=value;return vnone();}if(p->n==p->cap){size_t cap=p->cap?p->cap*2:8;Pair*a=mem(cap*sizeof(Pair));if(p->a){memcpy(a,p->a,p->n*sizeof(Pair));free(p->a);}p->a=a;p->cap=cap;}p->a[p->n].key=k;p->a[p->n++].value=value;return vnone();}
static V vmget(V m,V k,int has){Map*p=m.p;size_t i;for(i=0;i<p->n;i++)if(eq(p->a[i].key,k))return has?vb(1):vr(1,p->a[i].value);return has?vb(0):verr("map key not found");}
static V vread(V path){FILE*f=fopen(path.s,"rb");char*b;long n;size_t got;if(!f)return verr(strerror(errno));if(fseek(f,0,SEEK_END)!=0||(n=ftell(f))<0){fclose(f);return verr("Could not determine file size");}rewind(f);b=mem((size_t)n+1);got=fread(b,1,(size_t)n,f);if(ferror(f)){fclose(f);free(b);return verr("File read failed");}fclose(f);V v=vsn(b,got);free(b);return vr(1,v);}
static V vwrite(V path,V s){FILE*f=fopen(path.s,"wb");size_t n;int closed;if(!f)return verr(strerror(errno));n=fwrite(s.s,1,s.len,f);closed=fclose(f);if(n!=s.len||closed)return verr("File write failed");return vr(1,vb(1));}
static V vremove(V path){if(remove(path.s))return verr(strerror(errno));return vr(1,vb(1));}
static V vexists(V path){FILE*f=fopen(path.s,"rb");if(f){fclose(f);return vb(1);}return vb(0);}
static V vintparse(V s){char*end;long long n;errno=0;n=strtoll(s.s,&end,10);if(errno||end!=s.s+s.len||end==s.s)return verr("invalid integer");return vr(1,vi(n));}
static V vslice(V s,V a,V b){if(a.i<0||b.i<a.i||(uint64_t)b.i>s.len)return verr("string byte range out of bounds");return vr(1,vsn(s.s+a.i,(size_t)(b.i-a.i)));}
static V vobject(int n,const char**names,V*values){Object*p=mem(sizeof(*p));p->n=n;p->names=mem(sizeof(char*)*(n?n:1));p->values=mem(sizeof(V)*(n?n:1));memcpy(p->names,names,n*sizeof(char*));memcpy(p->values,values,n*sizeof(V));return vp(LIAF_OBJECT,p);}
static V vfield(V obj,const char*name){Object*p=obj.p;size_t i;for(i=0;i<p->n;i++)if(!strcmp(p->names[i],name))return p->values[i];return vnone();}
static V vdiv(V a,V b){if(a.tag==LIAF_FLOATING)return vf(a.f/b.f);if(!b.i){fputs("LIAF: integer division by zero\n",stderr);exit(2);}if(a.i==INT64_MIN&&b.i==-1)return a;return vi(a.i/b.i);}

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
