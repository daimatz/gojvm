import java.util.ArrayList;
import java.util.HashMap;

public class NestedLoop {
    public static void main(String[] args) {
        // Matrix multiplication (2x2)
        int[][] a = {{1, 2}, {3, 4}};
        int[][] b = {{5, 6}, {7, 8}};
        int[][] c = new int[2][2];
        for (int i = 0; i < 2; i++) {
            for (int j = 0; j < 2; j++) {
                for (int k = 0; k < 2; k++) {
                    c[i][j] += a[i][k] * b[k][j];
                }
            }
        }
        // c = {{19,22},{43,50}}
        System.out.println(c[0][0]); // 19
        System.out.println(c[0][1]); // 22
        System.out.println(c[1][0]); // 43
        System.out.println(c[1][1]); // 50

        // Word frequency count
        String[] words = {"apple", "banana", "apple", "cherry", "banana", "apple"};
        HashMap<String, Integer> freq = new HashMap<>();
        for (String w : words) {
            Integer count = freq.get(w);
            if (count == null) {
                freq.put(w, 1);
            } else {
                freq.put(w, count + 1);
            }
        }
        // Sort keys for deterministic output
        ArrayList<String> keys = new ArrayList<>(freq.keySet());
        java.util.Collections.sort(keys);
        for (String key : keys) {
            System.out.println(key + ":" + freq.get(key));
        }
    }
}
