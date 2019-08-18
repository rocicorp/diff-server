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
  private static final String CHANNEL = "replicant.dev/examples/todo";

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
            try {
              // TODO: Can we send from dart as bytes instead?
              byte[] data = (byte[])(MainActivity.this.getConnection().dispatch(call.method, ((String)call.arguments).getBytes()));
              result.success(new String(data));
            } catch (Exception e) {
              result.error("Bonk", e.toString(), null);
            }
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
    tmpDir.deleteOnExit();
    if (tmpDir.exists()) {
      tmpDir.delete();
    }
    if (!tmpDir.mkdirs()) {
      throw new RuntimeException("Could not make temp dir!");
    }
  }
}
