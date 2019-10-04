package dev.roci;

import android.content.Context;
import android.os.HandlerThread;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;

import com.facebook.react.bridge.Callback;
import com.facebook.react.bridge.ReactApplicationContext;
import com.facebook.react.bridge.ReactContextBaseJavaModule;
import com.facebook.react.bridge.ReactMethod;
import com.facebook.react.bridge.Promise;

import java.io.File;

public class ReplicantModule extends ReactContextBaseJavaModule {
  
  private final ReactApplicationContext reactContext;
  private final Handler uiThreadHandler;
  private final Handler generalHandler;
  private final Handler syncHandler;

  
  public ReplicantModule(ReactApplicationContext reactContext) {
    super(reactContext);
    this.reactContext = reactContext;

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
  public String getName() {
    return "Replicant";
  }
  
  @ReactMethod
  public void dispatch(final String dbName, final String method, final String arguments, final Promise promise) {
    Log.i("Replicant", "Calling: " + method + " with arguments: " + arguments);

    Handler handler;
    if (method.equals("sync")) {
      handler = syncHandler;
    } else {
      handler = generalHandler;
    }

    handler.post(new Runnable() {
      public void run() {
        // TODO: Avoid conversion here - can dart just send as bytes?
        byte[] argData = arguments.getBytes();
        byte[] resultData = null;
        Exception exception = null;
        try {
          resultData = repm.Repm.dispatch(dbName, method, argData);
        } catch (Exception e) {
          exception = e;
        }

        sendResult(promise, resultData, exception);
      }
    });
  }

  private void sendResult(final Promise promise, final byte[] data, final Exception e) {
    // TODO: Avoid conversion here - can dart accept bytes?
    final String retStr = data != null && data.length > 0 ? new String(data) : "";
    uiThreadHandler.post(new Runnable() {
      @Override
      public void run() {
        if (e != null) {
          promise.reject("Replicant error", e.toString());
        } else {
          promise.resolve(retStr);
        }
      }
    });
  }
  
  private void initReplicant() {
    File replicantDir = reactContext.getFileStreamPath("replicant");
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
