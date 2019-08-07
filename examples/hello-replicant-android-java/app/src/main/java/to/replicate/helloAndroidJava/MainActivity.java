package to.replicate.helloAndroidJava;

import androidx.appcompat.app.AppCompatActivity;

import android.app.AlertDialog;
import android.os.Bundle;
import android.util.Log;

import org.json.JSONObject;
import org.json.JSONTokener;

import java.io.File;

public class MainActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
    }

    @Override
    protected void onResume() {
        super.onResume();

        String message = "Error";
        try {
            File f = this.getFileStreamPath("db3");
            repm.Connection conn = repm.Repm.open(f.getAbsolutePath(), "client1");
            conn.dispatch("putBundle", "{\"code\": \"function setGreeting(greet) { db.put('greet', greet); }\"}".getBytes());
            conn.dispatch("exec", "{\"name\": \"setGreeting\", \"args\": [\"Aloha\"]}".getBytes());
            byte[] result = conn.dispatch("get", "{\"key\": \"greet\"}".getBytes());
            message = ((JSONObject)new JSONTokener(new String(result)).nextValue()).getString("data") + ", Replicant!";
        } catch (Exception e) {
            Log.e("blech", e.toString());
        }

        new AlertDialog.Builder(this)
                .setTitle("beep")
                .setMessage(message)
                .show();
    }

}
