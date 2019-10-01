package roci.dev.replicant;

import android.content.Context;
import android.os.HandlerThread;
import android.os.Handler;
import android.os.Looper;

import io.flutter.plugin.common.MethodCall;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugin.common.MethodChannel.MethodCallHandler;
import io.flutter.plugin.common.MethodChannel.Result;
import io.flutter.plugin.common.PluginRegistry.Registrar;

import java.io.File;
import java.util.ArrayList;

import android.util.Log;

/** ReplicantPlugin */
public class ReplicantPlugin implements MethodCallHandler {
  private static final String CHANNEL = "replicant.dev";
  private static Context appContext;

  private Handler uiThreadHandler;
  private Handler generalHandler;
  private Handler syncHandler;

  /** Plugin registration. */
  public static void registerWith(Registrar registrar) {
    appContext = registrar.context();
    final MethodChannel channel = new MethodChannel(registrar.messenger(), CHANNEL);
    channel.setMethodCallHandler(new ReplicantPlugin());
  }

  public ReplicantPlugin() {
    uiThreadHandler = new Handler(Looper.getMainLooper());

    // Most Replicant operations happen serially, but not blocking UI thread.
    HandlerThread generalThread = new HandlerThread("replicant.dev/general");
    generalThread.start();
    generalHandler = new Handler(generalThread.getLooper()); 

    // Sync shouldn't block the UI or other Replicant operations.
    HandlerThread syncThread = new HandlerThread("replicant.dev/sync");
    syncThread.start();
    syncHandler = new Handler(syncThread.getLooper()); 

    generalHandler.post(new Runnable() {
      public void run() {
        Log.i("Replicant", "init");
        initReplicant();
      }
    });
  }

  @Override
  public void onMethodCall(final MethodCall call, final Result result) {
    Log.i("Replicant", "Calling: " + call.method + " with arguments: " + (String)((ArrayList)call.arguments).get(1));

    Handler handler;
    if (call.method.equals("sync")) {
      handler = syncHandler;
    } else {
      handler = generalHandler;
    }

    handler.post(new Runnable() {
      public void run() {
        // The arguments passed from Flutter is a two-element array:
        // 0th element is the name of the database to call on
        // 1st element are the rpc arguments (JSON-encoded)
        ArrayList args = (ArrayList)call.arguments;

        String dbName = (String)args.get(0);
        // TODO: Avoid conversion here - can dart just send as bytes?
        byte[] argData = ((String)args.get(1)).getBytes();

        byte[] resultData = null;
        Exception exception = null;
        try {
          resultData = repm.Repm.dispatch(dbName, call.method, argData);
        } catch (Exception e) {
          exception = e;
        }

        sendResult(result, resultData, exception);
      }
    });
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

  private static void initReplicant() {
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
      repm.Repm.init(dataDir.getAbsolutePath(), tmpDir.getAbsolutePath());
    } catch (Exception e) {
      Log.e("Replicant", "Could not initialize Replicant", e);
    }
  }
}
