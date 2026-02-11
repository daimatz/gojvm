public class ArrayCopyTest {
    public static void main(String[] args) {
        // System.arraycopy
        int[] src = {1, 2, 3, 4, 5};
        int[] dst = new int[5];
        System.arraycopy(src, 0, dst, 0, 5);
        for (int i = 0; i < dst.length; i++) {
            System.out.println(dst[i]);
        }
        // 1 2 3 4 5

        // Partial copy
        int[] arr = {10, 20, 30, 40, 50};
        System.arraycopy(arr, 1, arr, 0, 4);
        // arr = {20, 30, 40, 50, 50}
        System.out.println(arr[0]); // 20
        System.out.println(arr[3]); // 50

        // Object array copy
        String[] names = {"Alice", "Bob", "Charlie"};
        String[] copy = new String[3];
        System.arraycopy(names, 0, copy, 0, 3);
        System.out.println(copy[1]); // Bob
    }
}
