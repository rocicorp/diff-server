package com.example.todo;

import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import io.flutter.app.FlutterActivity;
import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugin.common.MethodChannel.MethodCallHandler;
import io.flutter.plugin.common.MethodChannel.Result;
import io.flutter.plugins.GeneratedPluginRegistrant;

import java.io.File;
import java.util.Date;

import android.util.Log;

public class MainActivity extends FlutterActivity {
  private static final String CHANNEL = "replicant.dev/samples/todo";

  private static repm.Connection conn;

  private static File tmpDir;

  private Handler uiThreadHandler;

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);

    uiThreadHandler = new Handler(Looper.getMainLooper());

    GeneratedPluginRegistrant.registerWith(this);

    initTempDir();
    initConnection();
    if (conn == null) {
      return;
    }

    new MethodChannel(getFlutterView(), CHANNEL).setMethodCallHandler(
      new MethodCallHandler() {
          @Override
          public void onMethodCall(MethodCall call, Result result) {
            // TODO: Do we maybe not want to create a new thread for every call?
            // Tempting to use AsyncTask but I'm not sure how many threads the backing pool
            // has and don't want sync(), which can block for a long time, to block other
            // calls into Replicant which should be near-instant.
            new Thread(new Runnable() {
              public void run() {
                // TODO: Avoid conversion here - can dart just send as bytes?
                byte[] argData = new byte[0];
                if (call.arguments != null) {
                  argData = ((String)call.arguments).getBytes();
                }

                byte[] resultData = null;
                Exception exception = null;
                try {
                  resultData = conn.dispatch(call.method, argData);
                } catch (Exception e) {
                  exception = e;
                }

                sendResult(result, resultData, exception);
              }
            }).start();
          }
      }
    );
  }

  private void sendResult(Result result, final byte[] data, final Exception e) {
    // TODO: Avoid conversion here - can dart accept bytes?
    final String retStr = data != null && data.length > 0 ? new String(data) : "";
    uiThreadHandler.post(new Runnable() {
      @Override
      public void run() {
        if (e != null) {
          result.error("Replicant error", e.toString(), null);
        } else {
          result.success(retStr);
        }
      }
    });
  }

  private void initConnection() {
    try {
      if (MainActivity.conn == null) {
        File f = this.getFileStreamPath("db3");
        MainActivity.conn = repm.Repm.open(f.getAbsolutePath(), "client1", tmpDir.getAbsolutePath());
      }
    } catch (Exception e) {
      Log.e("Replicant", "Could not create connection: " + e.toString());
    }
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
