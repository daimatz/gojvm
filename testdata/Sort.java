class Sort {
    public static void sort(int[] a) {
        for (int i = 0; i < a.length - 1; i++)
            for (int j = 0; j < a.length - 1 - i; j++)
                if (a[j] > a[j + 1]) {
                    int t = a[j]; a[j] = a[j + 1]; a[j + 1] = t;
                }
    }
    public static void main(String[] args) {
        int[] a = {5, 3, 1, 4, 2};
        sort(a);
        for (int i = 0; i < a.length; i++)
            System.out.println(a[i]);
    }
}
