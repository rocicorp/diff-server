package roci.dev.replicant;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;

import io.flutter.app.FlutterActivity;
import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugin.common.MethodChannel.MethodCallHandler;
import io.flutter.plugin.common.MethodChannel.Result;
import io.flutter.plugin.common.PluginRegistry.Registrar;

import java.io.File;
import java.util.Date;
import java.util.UUID;

import android.util.Log;

/** ReplicantPlugin */
public class ReplicantPlugin implements MethodCallHandler {
  private static final String CHANNEL = "replicant.dev";
  private static repm.Connection conn;
  private static File tmpDir;
  private static String clientID;
  private static Context appContext;

  private Handler uiThreadHandler;

  /** Plugin registration. */
  public static void registerWith(Registrar registrar) {
    appContext = registrar.context();
    final MethodChannel channel = new MethodChannel(registrar.messenger(), CHANNEL);
    channel.setMethodCallHandler(new ReplicantPlugin());
  }

  public ReplicantPlugin() {
    uiThreadHandler = new Handler(Looper.getMainLooper());
  }

  @Override
  public void onMethodCall(final MethodCall call, final Result result) {
    // TODO: Do we maybe not want to create a new thread for every call?
    // Tempting to use AsyncTask but I'm not sure how many threads the backing pool
    // has and don't want sync(), which can block for a long time, to block other
    // calls into Replicant which should be near-instant.
    Log.i("Replicant", "Calling: " + call.method + " with arguments: " + (String)call.arguments);
    new Thread(new Runnable() {
      public void run() {
        ReplicantPlugin.this.initClientID();
        ReplicantPlugin.this.initConnection();

        if (conn == null) {
          sendResult(result, new byte[0], new Exception("Could not open Replicant database"));
          return;
        }

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

  private void sendResult(final Result result, final byte[] data, final Exception e) {
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

  private synchronized void initConnection() {
    if (ReplicantPlugin.conn != null) {
      return;
    }

    if (ReplicantPlugin.clientID == null) {
      Log.e("Replicant", "clientID is null, cannot open database");
      return;
    }

    File replicantDir = appContext.getFileStreamPath("replicant");
    File dataDir = new File(replicantDir, "data");
    File tmpDir = new File(replicantDir, "temp");

    // Android apps can't create directories in the global tmp directory, so we must create our own.
    if (!tmpDir.exists()) {
      if (!tmpDir.mkdirs()) {
        Log.e("Replicant", "Could not create temp directory");
        return;
      }
    }
    tmpDir.deleteOnExit();

    try {
      ReplicantPlugin.conn = repm.Repm.open(dataDir.getAbsolutePath(), ReplicantPlugin.clientID, tmpDir.getAbsolutePath());
    } catch (Exception e) {
      Log.e("Replicant", "Could not open Replicant database", e);
    }
  }

  private synchronized void initClientID() {
    if (ReplicantPlugin.clientID != null) {
      return;
    }

    SharedPreferences sharedPref = appContext.getSharedPreferences("replicant", Context.MODE_PRIVATE);
    ReplicantPlugin.clientID = sharedPref.getString("clientID", null);
    if (ReplicantPlugin.clientID != null) {
      return;
    }

    String cid = UUID.randomUUID().toString();
    SharedPreferences.Editor editor = sharedPref.edit();
    editor.putString("clientID", cid);
    editor.commit();
    Log.i("Replicant", "Generated and saved new clientID: " + cid);
    ReplicantPlugin.clientID = cid;
  }
}