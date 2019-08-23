package com.example.todo;

import android.os.Bundle;
import io.flutter.app.FlutterActivity;
import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugin.common.MethodChannel.MethodCallHandler;
import io.flutter.plugin.common.MethodChannel.Result;
import io.flutter.plugins.GeneratedPluginRegistrant;

import java.io.File;
import android.util.Log;

public class MainActivity extends FlutterActivity {
  private static final String CHANNEL = "replicant.dev/samples/todo";

  private static repm.Connection conn;

  private static File tmpDir;

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);

    GeneratedPluginRegistrant.registerWith(this);

    new MethodChannel(getFlutterView(), CHANNEL).setMethodCallHandler(
      new MethodCallHandler() {
          @Override
          public void onMethodCall(MethodCall call, Result result) {
            initTempDir();

            // TODO: Avoid conversion here.
            byte[] argData = new byte[0];
            if (call.arguments != null) {
              argData = ((String)call.arguments).getBytes();
            }

            byte[] data;
            try {
              data = MainActivity.this.getConnection().dispatch(call.method, argData);
            } catch (Exception e) {
              result.error("Replicant error", e.toString(), null);
              return;
            }

            // TODO: Avoid conversion here.
            String retStr = "";
            if (data != null && data.length > 0) {
              retStr = new String(data);
            }
            result.success(retStr);
          }
      }
    );
  }

  private repm.Connection getConnection() throws Exception {
    if (MainActivity.conn == null) {
      File f = this.getFileStreamPath("db3");
      MainActivity.conn = repm.Repm.open(f.getAbsolutePath(), "client1", tmpDir.getAbsolutePath());
    }
    return MainActivity.conn;
  }

  private void initTempDir() throws RuntimeException {
    if (tmpDir != null) {
      return;
    }

    tmpDir = new File(new File(getCacheDir(), "replicant"), "temp");
    if (!tmpDir.exists()) {
      if (!tmpDir.mkdirs()) {
        throw new RuntimeException("Could not make temp dir!");
      }
    }
    tmpDir.deleteOnExit();
  }
}
