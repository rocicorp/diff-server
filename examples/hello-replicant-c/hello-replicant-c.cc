#include "api.h"
#include <unistd.h>
#include <string>

// > go build -buildmode=c-archive -o replc.a ../../api/replc
// > clang++ *.cc replc.a -o hello-replicant-c -framework CoreFoundation -framework Security
// > ./hello-replicant-c
// Hello, from Replicant!

void checkErr(const std::string& desc, char* err) {
    if (err != NULL) {
        printf("%s error: %s\n", desc.c_str(), err);
        exit(1);
    }
}

void exec(int connID, const std::string& cmd, const std::string& in, std::string* out) {
    int execID = 0;
    checkErr("Exec", Exec(connID, const_cast<char*>(cmd.data()), cmd.size(), &execID));

    if (!in.empty()) {
        checkErr("ExecWrite", ExecWrite(execID, const_cast<char*>(in.data()), in.size()));
    }

    if (out != NULL) {
        int chunkSize = 1024;
        *out = std::string(1024, 0);
        for (int i = 0; ; i += chunkSize) {
            int readLen = 0;
            checkErr("ExecRead", ExecRead(execID, const_cast<char*>(&out->data()[i]), chunkSize, &readLen));
            if (readLen == 0) {
                break;
            }
            out->resize(out->size()+chunkSize);
        }
    }

    checkErr("ExecDone", ExecDone(execID));
}

int main() {
    const std::string dbSpec("/tmp/foo");
    int connID = 0;
    checkErr("Open", Open(const_cast<char*>(dbSpec.data()), dbSpec.size(), &connID));

    exec(connID, "{\"put\": {\"id\": \"obj1\"}}", "\"Hello, from Replicant!\"", NULL);

    std::string msg;
    exec(connID, "{\"get\": {\"id\": \"obj1\"}}", "", &msg);

    printf("%s\n", msg.c_str());
    return 0;
}
