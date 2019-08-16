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

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);
    GeneratedPluginRegistrant.registerWith(this);
    Log.d("Replicant", "hello");

    new MethodChannel(getFlutterView(), CHANNEL).setMethodCallHandler(
      new MethodCallHandler() {
          @Override
          public void onMethodCall(MethodCall call, Result result) {
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
      MainActivity.conn = repm.Repm.open(f.getAbsolutePath(), "client1");
    }
    return MainActivity.conn;
  }
}
