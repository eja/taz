#include <jni.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/ioctl.h>
#include <sys/wait.h>
#include <termios.h>
#include <dirent.h>
#include <errno.h>
#include <android/log.h>

#define LOG_TAG "EJA_JNI"
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, LOG_TAG, __VA_ARGS__)

#define JNI_METHOD __attribute__((visibility("default"))) JNIEXPORT

#ifdef __cplusplus
extern "C" {
#endif

static int create_subprocess_internal(char const* cmd, char const* cwd, char* const argv[], char** envp, int* pProcessId) {
    int ptm = open("/dev/ptmx", O_RDWR | O_CLOEXEC);
    if (ptm < 0) {
        LOGE("FAILED to open /dev/ptmx: %s", strerror(errno));
        return -1;
    }

    char* devname = ptsname(ptm);
    if (!devname) {
        LOGE("FAILED ptsname: %s", strerror(errno));
        close(ptm); 
        return -1; 
    }
    
    if (grantpt(ptm) || unlockpt(ptm)) {
        LOGE("FAILED grantpt/unlockpt: %s", strerror(errno));
        close(ptm); 
        return -1; 
    }

    pid_t pid = fork();
    if (pid < 0) {
        LOGE("FAILED to fork: %s", strerror(errno));
        close(ptm);
        return -1;
    } else if (pid > 0) {
        *pProcessId = (int) pid;
        return ptm;
    } else {
        close(ptm);
        setsid();
        
        int pts = open(devname, O_RDWR);
        if (pts < 0) {
            LOGE("CHILD: Failed to open PTS slave: %s", strerror(errno));
            exit(1);
        }

        dup2(pts, 0); // STDIN
        dup2(pts, 1); // STDOUT
        dup2(pts, 2); // STDERR

        DIR* self_dir = opendir("/proc/self/fd");
        if (self_dir != NULL) {
            int self_dir_fd = dirfd(self_dir);
            struct dirent* entry;
            while ((entry = readdir(self_dir)) != NULL) {
                int fd = atoi(entry->d_name);
                if (fd > 2 && fd != self_dir_fd) close(fd);
            }
            closedir(self_dir);
        }

        if (envp) {
            clearenv();
            for (; *envp; ++envp) putenv(*envp);
        }

        if (cwd) {
            if (chdir(cwd) != 0) {
                LOGE("CHILD: Failed to chdir to %s: %s", cwd, strerror(errno));
                exit(1);
            }
        }

        LOGD("CHILD: Attempting execvp: %s", cmd);
        
        execvp(cmd, argv);

        LOGE("CHILD: CRITICAL EXEC FAILURE '%s': %s (errno=%d)", cmd, strerror(errno), errno);
        
        exit(1);
    }
    return -1;
}

JNI_METHOD jint JNICALL Java_it_eja_taz_NativeLoader_createSubprocess(
        JNIEnv* env, jclass clazz,
        jstring cmd, jstring cwd, jobjectArray args, jobjectArray envVars, jintArray processIdArray) {

    const char* cmd_utf8 = (*env)->GetStringUTFChars(env, cmd, NULL);
    const char* cwd_utf8 = (*env)->GetStringUTFChars(env, cwd, NULL);
    
    jsize args_size = args ? (*env)->GetArrayLength(env, args) : 0;
    char** argv = (char**) malloc((args_size + 2) * sizeof(char*));
    argv[0] = strdup(cmd_utf8);
    for (int i = 0; i < args_size; ++i) {
        jstring arg_obj = (jstring) (*env)->GetObjectArrayElement(env, args, i);
        const char* arg_str = (*env)->GetStringUTFChars(env, arg_obj, NULL);
        argv[i + 1] = strdup(arg_str);
        (*env)->ReleaseStringUTFChars(env, arg_obj, arg_str);
    }
    argv[args_size + 1] = NULL;

    jsize env_size = envVars ? (*env)->GetArrayLength(env, envVars) : 0;
    char** envp = NULL;
    if (env_size > 0) {
        envp = (char**) malloc((env_size + 1) * sizeof(char*));
        for (int i = 0; i < env_size; ++i) {
            jstring env_obj = (jstring) (*env)->GetObjectArrayElement(env, envVars, i);
            const char* env_str = (*env)->GetStringUTFChars(env, env_obj, NULL);
            envp[i] = strdup(env_str);
            (*env)->ReleaseStringUTFChars(env, env_obj, env_str);
        }
        envp[env_size] = NULL;
    }

    int procId = 0;
    int ptm = create_subprocess_internal(cmd_utf8, cwd_utf8, argv, envp, &procId);

    (*env)->ReleaseStringUTFChars(env, cmd, cmd_utf8);
    (*env)->ReleaseStringUTFChars(env, cwd, cwd_utf8);

    for (int i = 0; argv[i] != NULL; i++) free(argv[i]);
    free(argv);
    if (envp) {
        for (int i = 0; envp[i] != NULL; i++) free(envp[i]);
        free(envp);
    }

    if (ptm >= 0) {
        int* pPid = (int*) (*env)->GetPrimitiveArrayCritical(env, processIdArray, NULL);
        if (pPid) {
            *pPid = procId;
            (*env)->ReleasePrimitiveArrayCritical(env, processIdArray, pPid, 0);
        }
    }

    return ptm;
}

#ifdef __cplusplus
}
#endif
